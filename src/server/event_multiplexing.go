package server

import (
	"syscall"
	"net"
	"sync"
	"time"
	"kiwi/src/server/event_internal"
	"runtime"
	"sync/atomic"
	"errors"
	"os"
	"github.com/kavu/go_reuseport"
)

var errClosing = errors.New("closing")
//var errCloseConns = errors.New("close conns")

func reusePortListen(proto, addr string) (l net.Listener, err error) {
	return reuseport.Listen(proto, addr)
}

type loop struct {
	idx    int             // loop index in the mpEventServer loops list
	poll   *internal.Poll  // epoll or kqueue
	buf    []byte          // read packet buffer
	fdclis map[int]*Client /// loop connections fd -> clients
	count  int32           // connection count
}

type conn struct {
	fd         int              // file descriptor
	lnidx      int              // listener index in the mpEventServer lns list
	loopidx    int              // owner loop
	out        []byte           // write buffer
	sa         syscall.Sockaddr // remote socket address
	reuse      bool             // should reuse input buffer
	opened     bool             // connection opened evio fired
	action     Action           // next user action
	ctx        interface{}      // user-defined context
	addrIndex  int
	localAddr  net.Addr
	remoteAddr net.Addr
}

func (c *conn) Context() interface{}       { return c.ctx }
func (c *conn) SetContext(ctx interface{}) { c.ctx = ctx }
func (c *conn) AddrIndex() int             { return c.addrIndex }
func (c *conn) LocalAddr() net.Addr        { return c.localAddr }
func (c *conn) RemoteAddr() net.Addr       { return c.remoteAddr }

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
type mpEventServer struct {
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
func (s *mpEventServer) waitForShutdown() {
	s.cond.L.Lock()
	s.cond.Wait()
	s.cond.L.Unlock()
}

// signalShutdown signals a shutdown an begins mpEventServer closing
func (s *mpEventServer) signalShutdown() {
	s.cond.L.Lock()
	s.cond.Signal()
	s.cond.L.Unlock()
}

func mpServe(events Events, listeners []*listener) error {
	numLoops := events.NumLoops
	if numLoops <= 0 {
		if numLoops == 0 {
			numLoops = 1
		} else {
			numLoops = runtime.NumCPU()
		}
	}
	mpes := &mpEventServer{}
	mpes.events = events
	mpes.lns = listeners
	mpes.cond = sync.NewCond(&sync.Mutex{})
	mpes.balance = events.LoadBalance
	mpes.tch = make(chan time.Duration)
	if mpes.events.Serving != nil {
		var es EventServer
		es.NumLoops = numLoops
		es.Addrs = make([]net.Addr, len(listeners))
		for i, ln := range mpes.lns {
			es.Addrs[i] = ln.lnaddr
		}
		action := mpes.events.Serving(es)
		switch action {
		case None:
		case Shutdown:
			return nil
		}
	}

	defer func() {
		// wait on a signal for shutdown
		mpes.waitForShutdown()
		// notify all loops to close by closing all listeners
		for _, l := range mpes.loops {
			l.poll.Trigger(errClosing)
		}
		// wait on all loops to complete reading events
		mpes.wg.Wait()

		// close loops and all outstanding connections
		for _, l := range mpes.loops {
			for _, c := range l.fdclis {
				loopCloseConn(mpes, l, c, nil)
			}
			l.poll.Close()
		}
	}()

	// create loops locally and bind the listeners.
	for i := 0; i < numLoops; i++ {
		l := &loop{
			idx:    i,
			poll:   internal.OpenPoll(),
			buf:    make([]byte, 0xFFFF),
			fdclis: make(map[int]*Client),
		}
		for _, ln := range mpes.lns {
			l.poll.AddRead(ln.fd)
		}
		mpes.loops = append(mpes.loops, l)
	}
	// start loops in background
	mpes.wg.Add(len(mpes.loops))
	for _, l := range mpes.loops {
		go loopRun(mpes, l)
	}
	return nil
}

func loopCloseConn(mpes *mpEventServer, l *loop, c *Client, err error) error {
	atomic.AddInt32(&l.count, -1)
	delete(l.fdclis, c.Conn.(*conn).fd)
	syscall.Close(c.Conn.(*conn).fd)
	if mpes.events.Closed != nil {
		switch mpes.events.Closed(c, err) {
		case None:
		case Shutdown:
			return errClosing
		}
	}
	return nil
}

func loopDetachConn(mpes *mpEventServer, l *loop, c *Client, err error) error {
	if mpes.events.Detached == nil {
		return loopCloseConn(mpes, l, c, err)
	}
	l.poll.ModDetach(c.Conn.(*conn).fd)
	atomic.AddInt32(&l.count, -1)
	delete(l.fdclis, c.Conn.(*conn).fd)
	if err := syscall.SetNonblock(c.Conn.(*conn).fd, false); err != nil {
		return err
	}
	switch mpes.events.Detached(c, &detachedConn{fd: c.Conn.(*conn).fd}) {
	case None:
	case Shutdown:
		return errClosing
	}
	return nil
}

func loopNote(mpes *mpEventServer, l *loop, note interface{}) error {
	var err error
	switch v := note.(type) {
	case time.Duration:
		delay, action := mpes.events.Tick()
		switch action {
		case None:
		case Shutdown:
			err = errClosing
		}
		mpes.tch <- delay
	case error: // shutdown
		err = v
	}
	return err
}

func loopTicker(mpes *mpEventServer, l *loop) {
	for {
		if err := l.poll.Trigger(time.Duration(0)); err != nil {
			break
		}
		time.Sleep(<-mpes.tch)
	}
}

func loopRun(mpes *mpEventServer, l *loop) {
	defer func() {
		//fmt.Println("-- loop stopped --", l.idx)
		mpes.signalShutdown()
		mpes.wg.Done()
	}()

	if l.idx == 0 && mpes.events.Tick != nil {
		go loopTicker(mpes, l)
	}

	//fmt.Println("-- loop started --", l.idx)
	l.poll.Wait(func(fd int, note interface{}) error {
		if fd == 0 {
			return loopNote(mpes, l, note)
		}
		c := l.fdclis[fd]
		conn := c.Conn.(*conn)
		switch {
		case c == nil:
			return loopAccept(mpes, l, fd)
		case !conn.opened:
			return loopOpened(mpes, l, c)
		case len(conn.out) > 0:
			return loopWrite(mpes, l, c)
		case conn.action != None:
			return loopAction(mpes, l, c)
		default:
			return loopRead(mpes, l, c)
		}
	})
}

func loopAccept(mpes *mpEventServer, l *loop, fd int) error {
	for i, ln := range mpes.lns {
		if ln.fd == fd {
			if len(mpes.loops) > 1 {
				switch mpes.balance {
				case LeastConnections:
					n := atomic.LoadInt32(&l.count)
					for _, lp := range mpes.loops {
						if lp.idx != l.idx {
							if atomic.LoadInt32(&lp.count) < n {
								return nil // do not accept
							}
						}
					}
				case RoundRobin:
					if int(atomic.LoadUintptr(&mpes.accepted))%len(mpes.loops) != l.idx {
						return nil // do not accept
					}
					atomic.AddUintptr(&mpes.accepted, 1)
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
			c := CreateClient(conn, flag)
			l.fdclis[conn.fd] = c
			l.poll.AddReadWrite(conn.fd)
			atomic.AddInt32(&l.count, 1)
			break
		}
	}
	return nil
}

func loopOpened(mpes *mpEventServer, l *loop, c *Client) error {
	conn := c.Conn.(*conn)
	conn.opened = true
	conn.addrIndex = conn.lnidx
	conn.remoteAddr = internal.SockaddrToAddr(conn.sa)
	if mpes.events.Opened != nil {
		out, opts, action := mpes.events.Opened(c)
		if len(out) > 0 {
			conn.out = append([]byte{}, out...)
		}
		conn.action = action
		conn.reuse = opts.ReuseInputBuffer
		if opts.TCPKeepAlive > 0 {
			if _, ok := mpes.lns[conn.lnidx].ln.(*net.TCPListener); ok {
				internal.SetKeepAlive(conn.fd, int(opts.TCPKeepAlive/time.Second))
			}
		}
	}
	if len(conn.out) > 0 && conn.action == None {
		l.poll.ModRead(conn.fd)
	}
	return nil
}

func loopAction(mpes *mpEventServer, l *loop, c *Client) error {
	conn := c.Conn.(*conn)
	switch conn.action {
	default:
		conn.action = None
	case Close:
		return loopCloseConn(mpes, l, c, nil)
	case Shutdown:
		return errClosing
	case Detach:
		return loopDetachConn(mpes, l, c, nil)
	}
	if len(conn.out) == 0 && conn.action == None {
		l.poll.ModRead(conn.fd)
	}
	return nil
}

func loopRead(mpes *mpEventServer, l *loop, c *Client) error {
	conn := c.Conn.(*conn)
	var in []byte
	n, err := syscall.Read(conn.fd, l.buf)
	if n == 0 || err != nil {
		if err == syscall.EAGAIN {
			return nil
		}
		return loopCloseConn(mpes, l, c, err)
	}
	in = l.buf[:n]
	if !conn.reuse {
		in = append([]byte{}, in...)
	}
	if mpes.events.Data != nil {
		out, action := mpes.events.Data(c, in)
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

func loopWrite(mpes *mpEventServer, l *loop, c *Client) error {
	conn := c.Conn.(*conn)
	if mpes.events.PreWrite != nil {
		mpes.events.PreWrite()
	}
	n, err := syscall.Write(conn.fd, conn.out)
	if err != nil {
		if err == syscall.EAGAIN {
			return nil
		}
		return loopCloseConn(mpes, l, c, err)
	}
	if n == len(conn.out) {
		conn.out = nil
	} else {
		conn.out = conn.out[n:]
	}
	if len(conn.out) == 0 && conn.action == None {
		l.poll.ModRead(conn.fd)
	}
	return nil
}

