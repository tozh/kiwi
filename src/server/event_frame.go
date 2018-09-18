package server

import (
	"time"
	"net"
	"io"
	"os"
	"strings"
)

// Action is an action that occurs after the completion of an evio.
type Action int

const (
	// None indicates that no action should occur following an evio.
	None Action = iota
	// Detach detaches a connection. Not available for UDP connections.
	Detach
	// Close closes the connection.
	Close
	// Shutdown shutdowns the mpEventServer.
	Shutdown
)

// LoadBalance sets the load balancing method.
type LoadBalance int

const (
	// Random requests that connections are randomly distributed.
	Random LoadBalance = iota
	// RoundRobin requests that connections are distributed to a loop in a
	// round-robin fashion.
	RoundRobin
	// LeastConnections assigns the next accepted connection to the loop with
	// the least number of active connections.
	LeastConnections
)

// Options are set when the client opens.
type Options struct {
	// TCPKeepAlive (SO_KEEPALIVE) socket option.
	TCPKeepAlive time.Duration
	// ReuseInputBuffer will forces the connection to share and reuse the
	// same input packet buffer with all other connections that also use
	// this option.
	// Default value is false, which means that all input data which is
	// passed to the Data evio will be a uniquely copied []byte slice.
	ReuseInputBuffer bool
}

type EventServer struct {
	Addrs    []net.Addr
	NumLoops int
}

type addrOpts struct {
	reusePort bool
}

type Conn interface {
	// Context returns a user-defined context.
	Context() interface{}
	// SetContext sets a user-defined context.
	SetContext(interface{})
	// AddrIndex is the index of mpEventServer address that was passed to the Serve call.
	AddrIndex() int
	// LocalAddr is the connection's local socket address.
	LocalAddr() net.Addr
	// RemoteAddr is the connection's remote peer address.
	RemoteAddr() net.Addr
}

type Client interface {
	GetConn() Conn
}

type Events struct {
	// NumLoops sets the number of loops to use for the mpEventServer. Setting this
	// to a value greater than 1 will effectively make the mpEventServer
	// multithreaded for multi-core machines. Which means you must take care
	// with synchonizing memory between all evio callbacks. Setting to 0 or 1
	// will run the mpEventServer single-threaded. Setting to -1 will automatically
	// assign this value equal to runtime.NumProcs().
	NumLoops int
	// LoadBalance sets the load balancing method. Load balancing is always a
	// best effort to attempt to distribute the incoming connections between
	// multiple loops. This option is only works when NumLoops is set.
	LoadBalance LoadBalance

	Serving func(es EventServer) (action Action)

	// YOU HAVE TO DEFINCE THIS
	Accepted func(conn Conn, flags int) (c Client, action Action)

	// Opened fires when a new connection has opened.
	// The info parameter has information about the connection such as
	// it's local and remote address.
	// Use the out return value to write data to the connection.
	// The opts return value is used to set connection options.
	Opened func(c Client) (out []byte, opts Options, action Action)

	// Closed fires when a connection has closed.
	// The err parameter is the last known connection error.
	Closed func(c Client, err error) (action Action)
	// Detached fires when a connection has been previously detached.
	// Once detached it's up to the receiver of this evio to manage the
	// state of the connection. The Closed evio will not be called for
	// this connection.
	// The conn parameter is a ReadWriteCloser that represents the
	// underlying socket connection. It can be freely used in goroutines
	// and should be closed when it's no longer needed.
	Detached func(c Client, rwc io.ReadWriteCloser) (action Action)
	// PreWrite fires just before any data is written to any client socket.
	PreWrite func()
	// Data fires when a connection sends the mpEventServer data.
	// The in parameter is the incoming data.
	// Use the out return value to write data to the connection.
	Data func(c Client, in []byte) (out []byte, action Action)
	// Written fires after sending data
	// n is the length of conn.out
	Written func(c Client, n int) (action Action)
	// Tick fires immediately after the mpEventServer starts and will fire again
	// following the duration specified by the delay return value.
	Tick func() (delay time.Duration, action Action)
	// when the EventServer shutdown, Shutdown() fires
	Shutdown func()
}

// Serve starts handling events for the specified addresses.
//
// Addresses should use a scheme prefix and be formatted
// like `tcp://192.168.0.10:9851` or `unix://socket`.
// Valid network schemes:
//  tcp   - bind to both IPv4 and IPv6
//  tcp4  - IPv4
//  tcp6  - IPv6
//  unix  - Unix Domain Socket
//
// The "tcp" network scheme is assumed when one is not specified.

func EventServe(events Events, addr ...string) error {
	kiwiS.wg.Add(1)
	defer kiwiS.wg.Done()
	var lns []*listener
	defer func() {
		for _, ln := range lns {
			ln.close()
		}
	}()
	for _, addr := range addr {
		var ln listener
		ln.network, ln.addr, ln.opts = parseAddr(addr)
		if ln.network == "unix" {
			os.RemoveAll(ln.addr)
		}
		var err error
		if ln.opts.reusePort {
			// if reuse ports
			ln.ln, err = reusePortListen(ln.network, ln.addr)
		} else {
			ln.ln, err = net.Listen(ln.network, ln.addr)
		}
		if err != nil {
			return err
		}
		ln.lnaddr = ln.ln.Addr()
		if err := ln.system(); err != nil {
			return err
		}
		lns = append(lns, &ln)
	}
	return mpServe(events, lns)
}

func parseAddr(addr string) (network string, address string, opts addrOpts) {
	network = "tcp"
	address = addr
	opts.reusePort = false
	if strings.Contains(address, "://") {
		network = strings.Split(address, "://")[0]
		address = strings.Split(address, "://")[1]
	}
	q := strings.Index(address, "?")
	if q != -1 {
		for _, part := range strings.Split(address[q+1:], "&") {
			kv := strings.Split(part, "=")
			if len(kv) == 2 {
				switch kv[0] {
				case "reuseport":
					if len(kv[1]) != 0 {
						switch kv[1][0] {
						default:
							opts.reusePort = kv[1][0] >= '1' && kv[1][0] <= '9'
						case 'T', 't', 'Y', 'y':
							opts.reusePort = true
						}
					}
				}
			}
		}
		address = address[:q]
	}
	return
}
