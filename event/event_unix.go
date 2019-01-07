package event

import (
	"errors"
	"fmt"
	"github.com/kavu/go_reuseport"
	"github.com/zhaotong0312/kiwi/event/event_internal"
	"github.com/zhaotong0312/kiwi/server"
	"net"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)


var errClosing = errors.New("closing")
//var errCloseConns = errors.New("close conns")

func reusePortListen(proto, addr string) (l net.Listener, err error) {
	return reuseport.Listen(proto, addr)
}

type loop struct {
	idx    int            // loop index in the EventServer loops list
	poll   *internal.Poll // epoll or kqueue
	buf    []byte         // read packet buffer
	fdclis map[int]Client // loop connections fd -> clients
	count  int32          // connection count
}


type conn struct {
	fd         int              // file descriptor
	lnidx      int              // listener index in the server lns list
	out        []byte           // write buffer
	sa         syscall.Sockaddr // remote socket address
	reuse      bool             // should reuse input buffer
	opened     bool             // connection opened event fired
	action     Action           // next user action
	ctx        interface{}      // user-defined context
	addrIndex  int              // index of listening address
	localAddr  net.Addr         // local addre
	remoteAddr net.Addr         // remote addr
	loop       *loop            // connected loop
}

func (c *conn) Context() interface{}       { return c.ctx }
func (c *conn) SetContext(ctx interface{}) { c.ctx = ctx }
func (c *conn) AddrIndex() int             { return c.addrIndex }
func (c *conn) LocalAddr() net.Addr        { return c.localAddr }
func (c *conn) RemoteAddr() net.Addr       { return c.remoteAddr }
func (c *conn) Wake() {
	if c.loop != nil {
		c.loop.poll.Trigger(c)
	}
}

type detachedConn struct {
	fd int
}

func (c *detachedConn) Close() error {
	err := syscall.Close(c.fd)
	if err != nil {
		return err
	}
	c.fd = -1
	return nil
}

func (c *detachedConn) Read(p []byte) (n int, err error) {
	return syscall.Read(c.fd, p)
}

func (c *detachedConn) Write(p []byte) (n int, err error) {
	n = len(p)
	for len(p) > 0 {
		nn, err := syscall.Write(c.fd, p)
		if err != nil {
			return n, err
		}
		p = p[nn:]
	}
	return n, nil
}

type listener struct {
	ln      net.Listener
	lnaddr  net.Addr
	opts    addrOpts
	f       *os.File
	fd      int
	network string
	addr    string
}

func (ln *listener) close() {
	if ln.fd != 0 {
		syscall.Close(ln.fd)
	}
	if ln.f != nil {
		ln.f.Close()
	}
	if ln.ln != nil {
		ln.ln.Close()
	}
	if ln.network == "unix" {
		os.RemoveAll(ln.addr)
	}
}

// system takes the net listener and detaches it from it's parent
// evio loop, grabs the file descriptor, and makes it non-blocking.
func (ln *listener) system() error {
	var err error
	switch netln := ln.ln.(type) {
	case nil:
	case *net.TCPListener:
		ln.f, err = netln.File()
	case *net.UnixListener:
		ln.f, err = netln.File()
	}
	if err != nil {
		ln.close()
		return err
	}
	ln.fd = int(ln.f.Fd())
	return syscall.SetNonblock(ln.fd, true)
}

// EventServer implemented based on multiplexing of unix system
type EventServer struct {
	// The addrs parameter is an array of listening addresses that align
	// with the addr strings passed to the Serve function.
	Addrs []net.Addr
	// NumLoops is the number of loops that the server is using.
	NumLoops int

	events   Events             // user events
	loops    []*loop            // all the loops
	lns      []*listener        // all the listeners
	wg       sync.WaitGroup     // loop close waitgroup
	cond     *sync.Cond         // shutdown signaler
	balance  LoadBalance        // load balancing method
	accepted uintptr            // accept counter
	tch      chan time.Duration // ticker channel
}

// waitForShutdown waits for a signal to shutdown
func (es *EventServer) waitForShutdown() {
	es.cond.L.Lock()
	es.cond.Wait()
	es.cond.L.Unlock()
}

// signalShutdown signals a shutdown an begins EventServer closing
func (es *EventServer) signalShutdown() {
	es.cond.L.Lock()
	es.cond.Signal()
	es.cond.L.Unlock()
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
func CreateEventServer(events Events, addrs ...string) (*EventServer, error) {

	var lns []*listener
	for _, addr := range addrs {
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
			return nil, err
		}
		ln.lnaddr = ln.ln.Addr()
		if err := ln.system(); err != nil {
			return nil, err
		}
		lns = append(lns, &ln)
	}

	if events.NumLoops <= 0 {
		if events.NumLoops == 0 {
			events.NumLoops = 1
		} else {
			events.NumLoops = runtime.NumCPU()
		}
	}
	es := &EventServer{}
	es.events = events
	es.lns = lns
	es.cond = sync.NewCond(&sync.Mutex{})
	es.balance = events.LoadBalance
	es.tch = make(chan time.Duration)
	return es, nil
}

func Serve(es *EventServer) error {

	fmt.Println("numLoops---->", es.events.NumLoops)
	if es.events.Serving != nil {
		var s server.Server
		s.NumLoops = es.events.NumLoops
		s.Addrs = make([]net.Addr, len(es.lns))
		for i, ln := range es.lns {
			es.Addrs[i] = ln.lnaddr
		}
		action := es.events.Serving(es)
		switch action {
		case None:
		case Shutdown:
			return nil
		}
	}

	defer func() {
		for _, ln := range es.lns {
			fmt.Println("Closing Listeners at", ln.addr, "... Finished.")
			ln.close()
		}
	}()

	defer func() {
		// wait on a signal for shutdown
		es.waitForShutdown()
		// notify all loops to close by closing all listeners
		for _, l := range es.loops {
			l.poll.Trigger(errClosing)
		}
		// wait on all loops to complete reading events
		es.wg.Wait()

		// close loops and all outstanding connections
		for _, l := range es.loops {
			for _, c := range l.fdclis {
				loopCloseConn(es, l, c, nil)
			}
			l.poll.Close()
		}
		// do Shutdown action
		es.events.Shutdown()
	}()

	// create loops locally and bind the listeners.
	for i := 0; i < es.events.NumLoops; i++ {
		l := &loop{
			idx:    i,
			poll:   internal.OpenPoll(),
			buf:    make([]byte, 0xFFFF),
			fdclis: make(map[int]Client),
		}
		for _, ln := range es.lns {
			l.poll.AddRead(ln.fd)
		}
		es.loops = append(es.loops, l)
	}
	// start loops in background
	es.wg.Add(len(es.loops))
	for _, l := range es.loops {
		go loopRun(es, l)
	}
	return nil
}

func loopCloseConn(es *EventServer, l *loop, c Client, err error) error {
	conn := c.GetConn()
	atomic.AddInt32(&l.count, -1)
	delete(l.fdclis, conn.fd)
	syscall.Close(conn.fd)
	if es.events.Closed != nil {
		switch es.events.Closed(c, err) {
		case None:
		case Shutdown:
			return errClosing
		}
	}
	return nil
}

func loopDetachConn(es *EventServer, l *loop, c Client, err error) error {
	conn := c.GetConn()
	if es.events.Detached == nil {
		return loopCloseConn(es, l, c, err)
	}
	l.poll.ModDetach(conn.fd)
	atomic.AddInt32(&l.count, -1)
	delete(l.fdclis, conn.fd)
	if err := syscall.SetNonblock(conn.fd, false); err != nil {
		return err
	}
	switch es.events.Detached(c, &detachedConn{fd: conn.fd}) {
	case None:
	case Shutdown:
		return errClosing
	}
	return nil
}

func loopNote(es *EventServer, l *loop, note interface{}) error {
	var err error
	switch v := note.(type) {
	case time.Duration:
		delay, action := es.events.Tick()
		switch action {
		case None:
		case Shutdown:
			err = errClosing
		}
		es.tch <- delay
	case error: // shutdown
		err = v
	}
	return err
}

func loopTicker(es *EventServer, l *loop) {
	for {
		if err := l.poll.Trigger(time.Duration(0)); err != nil {
			break
		}
		time.Sleep(<-es.tch)
	}
}

func loopRun(es *EventServer, l *loop) {
	defer func() {
		//fmt.Println("-- loop stopped --", l.idx)
		es.signalShutdown()
		es.wg.Done()
	}()

	if l.idx == 0 && es.events.Tick != nil {
		go loopTicker(es, l)
	}

	//fmt.Println("-- loop started --", l.idx)
	l.poll.Wait(func(fd int, note interface{}) error {
		if fd == 0 {
			return loopNote(es, l, note)
		}
		c := l.fdclis[fd]
		switch {
		case c == nil:
			return loopAccept(es, l, fd)
		case !c.GetConn().opened:
			return loopOpened(es, l, c)
		case len(c.GetConn().out) > 0:
			return loopWrite(es, l, c)
		case c.GetConn().action != None:
			return loopAction(es, l, c)
		default:
			return loopRead(es, l, c)
		}
	})
}

func loopAccept(es *EventServer, l *loop, fd int) error {
	//fmt.Println("loopAccept")
	for i, ln := range es.lns {
		if ln.fd == fd {
			if len(es.loops) > 1 {
				switch es.balance {
				case LeastConnections:
					n := atomic.LoadInt32(&l.count)
					for _, lp := range es.loops {
						if lp.idx != l.idx {
							if atomic.LoadInt32(&lp.count) < n {
								return nil // do not accept
							}
						}
					}
				case RoundRobin:
					if int(atomic.LoadUintptr(&es.accepted))%len(es.loops) != l.idx {
						return nil // do not accept
					}
					atomic.AddUintptr(&es.accepted, 1)
				}
			}
			nfd, sa, err := syscall.Accept(fd)
			if err != nil {
				if err == syscall.EAGAIN {
					return nil
				}
				return err
			}
			if err := syscall.SetNonblock(nfd, true); err != nil {
				return err
			}
			conn := &conn{fd: nfd, sa: sa, lnidx: i}
			flag := 0
			if ln.network == "unix" {
				flag |= CLIENT_UNIX_SOCKET
			}
			l.poll.AddReadWrite(conn.fd)
			c, action := es.events.Accepted(conn, flag)
			if c == nil || action != None {
				return err
				conn.action = action
			}
			l.fdclis[conn.fd] = c
			atomic.AddInt32(&l.count, 1)
			break
		}
	}
	return nil
}

func loopOpened(es *EventServer, l *loop, c Client) error {
	conn := c.GetConn()
	conn.opened = true
	conn.addrIndex = conn.lnidx
	conn.remoteAddr = internal.SockaddrToAddr(conn.sa)
	if es.events.Opened != nil {
		out, opts, action := es.events.Opened(c)
		if len(out) > 0 {
			conn.out = append([]byte{}, out...)
		}
		conn.action = action
		conn.reuse = opts.ReuseInputBuffer
		if opts.TCPKeepAlive > 0 {
			if _, ok := es.lns[conn.lnidx].ln.(*net.TCPListener); ok {
				internal.SetKeepAlive(conn.fd, int(opts.TCPKeepAlive/time.Second))
			}
		}
	}
	if len(conn.out) > 0 && conn.action == None {
		l.poll.ModRead(conn.fd)
	}
	return nil
}

func loopAction(es *EventServer, l *loop, c Client) error {
	conn := c.GetConn()
	switch conn.action {
	default:
		conn.action = None
	case Close:
		return loopCloseConn(es, l, c, nil)
	case Shutdown:
		return errClosing
	case Detach:
		return loopDetachConn(es, l, c, nil)
	}
	if len(conn.out) == 0 && conn.action == None {
		l.poll.ModRead(conn.fd)
	}
	return nil
}

func loopRead(es *EventServer, l *loop, c Client) error {
	conn := c.GetConn()
	var in []byte
	n, err := syscall.Read(conn.fd, l.buf)
	if n == 0 || err != nil {
		if err == syscall.EAGAIN {
			return nil
		}
		return loopCloseConn(es, l, c, err)
	}
	in = l.buf[:n]
	if !conn.reuse {
		in = append([]byte{}, in...)
	}
	if es.events.Data != nil {
		out, action := es.events.Data(c, in)
		conn.action = action
		if len(out) > 0 {
			conn.out = append([]byte{}, out...)
		}
	}
	if len(conn.out) != 0 || conn.action != None {
		l.poll.ModReadWrite(conn.fd)
	}
	return nil
}

func loopWrite(es *EventServer, l *loop, c Client) error {
	conn := c.GetConn()
	if es.events.PreWrite != nil {
		es.events.PreWrite()
	}
	n, err := syscall.Write(conn.fd, conn.out)
	if err != nil {
		if err == syscall.EAGAIN {
			return nil
		}
		return loopCloseConn(es, l, c, err)
	}
	if n == len(conn.out) {
		conn.out = nil
	} else {
		conn.out = conn.out[n:]
	}
	if es.events.Written != nil {
		conn.action = es.events.Written(c, n)
	}
	if len(conn.out) == 0 && conn.action == None {
		l.poll.ModRead(conn.fd)
	}
	return nil
}
