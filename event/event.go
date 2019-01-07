package event

import (
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

// Action is an action that occurs after the completion of an evio.
type Action int

const (
	None Action = iota
	Detach
	Close
	Shutdown
)

// LoadBalance sets the load balancing method.
type LoadBalance int

const (
	Random LoadBalance = iota
	RoundRobin
	LeastConnections
)

// Options are set when the client opens.
type Options struct {
	TCPKeepAlive time.Duration
	ReuseInputBuffer bool
}

type addrOpts struct {
	reusePort bool
}

type Conn interface {
	Context() interface{}
	SetContext(interface{})
	AddrIndex() int
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	Wake()
}

type Client interface {
	GetConn() Conn
}

type Events struct {
	NumLoops int
	LoadBalance LoadBalance
	Serving func(es *EventServer) (action Action)

	Accepted func(conn Conn, flags int) (c Client, action Action)

	Opened func(c Client) (out []byte, opts Options, action Action)

	Closed func(c Client, err error) (action Action)

	Detached func(c Client, rwc io.ReadWriteCloser) (action Action)

	PreWrite func()

	Data func(c Client, in []byte) (out []byte, action Action)

	Written func(c Client, n int) (action Action)

	Tick func() (delay time.Duration, action Action)

	Shutdown func()
}

func parseAddr(addr string) (network string, address string, opts addrOpts) {
	fmt.Println("parseAddr")
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
	fmt.Println(network, address, opts)
	return
}
