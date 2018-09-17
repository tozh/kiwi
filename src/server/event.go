package server

import (
	"net"
	"time"
	"sync/atomic"
	"fmt"
)

type accepted struct {
	conn net.Conn
	err  error
}

func UnixServer(s *Server) {
	// fmt.Println("------>UnixServer")
	s.wg.Add(1)
	defer s.wg.Done()
	listener := AnetListenUnix(s.UnixSocketPath)
	if listener == nil {
		return
	}
	for {
		ch := make(chan accepted, 1)
		go func() {
			conn, err := listener.Accept()
			ch <- accepted{conn, err}
		}()
		select {
		case <-s.CloseCh:
			// s.ServerLogDebugF("-->%v\n", "UnixServer ------ SHUTDOWN")
			listener.Close()
			return
		case acc := <-ch:
			if acc.err != nil {
				// s.ServerLogDebugF("-->%v\n", "UnixServer ------ Accept Error")
				AnetSetErrorFormat("Unix Accept error: %s", acc.err)
				continue
			}
			// s.ServerLogDebugF("-->%v\n", "UnixServer ------ CommonServer")
			CommonServer(s, acc.conn, CLIENT_UNIX_SOCKET, "")
		}
	}
}

func TcpServer(s *Server, ip string) {
	// fmt.Println("------>TcpServer")
	s.wg.Add(1)
	defer s.wg.Done()
	listener := AnetListenTcp("tcp", ip, s.Port)
	defer listener.Close()
	if listener == nil {
		return
	}
	for {
		ch := make(chan accepted, 1)
		go func() {
			conn, err := listener.Accept()
			ch <- accepted{conn, err}
		}()
		select {
		case <-s.CloseCh:
			return
		case acc := <-ch:
			if acc.err != nil {
				continue
			}
			CommonServer(s, acc.conn, 0, ip)
		}
	}
}

func CommonServer(s *Server, conn net.Conn, flags int, ip string) {
	c := CreateClient(s, conn, flags)
	if c == nil {
		conn.Close()
		CloseClient(s, c)
	}
	if s.Clients.ListLength() > s.MaxClients {
		AddReplyError(s, c, "max number of clients reached")
		WriteToClient(s, c)
		CloseClient(s, c)
		atomic.AddInt64(&s.StatRejectedConn, 1)
	}
	if s.ProtectedMode && s.BindAddrCount == 0 && s.RequirePassword == nil && flags&CLIENT_UNIX_SOCKET == 0 && ip != "" {
		err := "-DENIED Redis is running in protected mode."
		AddReplyError(s, c, err)
		CloseClient(s, c)
		atomic.AddInt64(&s.StatRejectedConn, 1)
	}
	go ProcessQueriesLoop(s, c)
	go CloseClientListener(s, c)
}

func ProcessQueriesLoop(s *Server, c *Client) {
	s.wg.Add(1)
	defer s.wg.Done()
	for {
		processingCh := make(chan int, 1)
		go ProcessQuery(s, c, processingCh)
		select {
		case <-c.CloseCh:
			// server closed, broadcast
			close(processingCh)
			return
		case <-processingCh:
			// query processing finished
		}
	}
}

func HeartBeating(s *Server, c *Client, readCh chan int) {
	// fmt.Println("HeartBeatLoop")
	s.wg.Add(1)
	defer s.wg.Done()
	select {
	case <-c.CloseCh:
		return
	case <-readCh:
		return
	case <-time.After(c.MaxIdleTime):
		fmt.Println("HearBeat fail. 3s reached.")
		close(readCh)
		BroadcastCloseClient(c)
		return
	}
}

func BroadcastCloseClient(c *Client) {
	close(c.CloseCh)
}

func BroadcastCloseServer(s *Server) {
	// fmt.Println("BroadcastCloseServer")
	close(s.CloseCh)
}

func CloseClientListener(s *Server, c *Client) {
	s.wg.Add(1)
	defer s.wg.Done()
	select {
	case <-c.CloseCh:
		CloseClient(s, c)
	}
}
