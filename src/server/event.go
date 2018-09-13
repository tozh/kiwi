package server

import (
	"net"
	"fmt"
	"time"
)

type accepted struct {
	conn net.Conn
	err  error
}

func UnixServer(s *Server) {
	fmt.Println("------>UnixServer")
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
			s.ServerLogDebugF("-->%v\n", "UnixServer ------ SHUTDOWN")
			listener.Close()
			return
		case acc := <-ch:
			if acc.err != nil {
				s.ServerLogDebugF("-->%v\n", "UnixServer ------ Accept Error")
				AnetSetErrorFormat("Unix Accept error: %s", acc.err)
				continue
			}
			s.ServerLogDebugF("-->%v\n", "UnixServer ------ CommonServer")
			CommonServer(s, acc.conn, CLIENT_UNIX_SOCKET, "")
		}
	}
}

func TcpServer(s *Server, ip string) {
	fmt.Println("------>TcpServer")
	s.wg.Add(1)
	defer s.wg.Done()
	listener := AnetListenTcp("tcp", ip, s.Port)
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
			s.ServerLogDebugF("-->%v\n", "TcpServer ------ SHUTDOWN")
			listener.Close()
			return
		case acc := <-ch:
			if acc.err != nil {
				s.ServerLogDebugF("-->%v\n", "TcpServer ------ Accept Error")
				AnetSetErrorFormat("Tcp Accept error: %s", acc.err)
				continue
			}
			s.ServerLogDebugF("-->%v\n", "TcpServer ------ CommonServer")
			CommonServer(s, acc.conn, 0, ip)
		}
	}
}

func CommonServer(s *Server, conn net.Conn, flags int64, ip string) {
	c := CreateClient(s, conn, flags)
	if c == nil {
		conn.Close()
		CloseClient(s, c)
	}
	if s.Clients.ListLength() > s.MaxClients {
		err := []byte("-ERR max number of clients reached\r\n")
		conn.Write(err)
		s.StatRejectedConn++
		CloseClient(s, c)
	}

	if s.ProtectedMode && s.BindAddrCount == 0 && s.RequirePassword == nil && flags&CLIENT_UNIX_SOCKET == 0 && ip != "" {
		err := []byte(
			`-DENIED Redis is running in protected mode because protected mode is enabled, no bind address was specified, no authentication password is requested to clients. In this mode 
connections are only accepted from the loopback interface. 

If you want to connect from external computers to Redis you may adopt one of the following solutions: 

1) Just disable protected mode sending the command 'CONFIG SET protected-mode no' from the loopback interface by connecting to Redis from the same host the test_server is running, however MAKE SURE Redis is not publicly accessible from internet if you do so. Use CONFIG REWRITE to make this change permanent.
2) Alternatively you can just disable the protected mode by editing the Redis configuration file, and setting the protectedmode option to 'no', and then restarting the test_server.
3) If you started the test_server manually just for testing, restart it with the '--protected-mode no' option.
4) Setup a bind address or an authentication password. 

NOTE: You only need to do one of the above things in order for the test_server to start accepting connections from the outside.\r\n`)
		conn.Write(err)
		s.StatRejectedConn++
	}
	go ProcessClientLoop(s, c)
	go CloseClientListener(s, c)
}

func ProcessClientLoop(s *Server, c *Client) {
	fmt.Println("ReadLoop")
	s.wg.Add(1)
	defer s.wg.Done()
	for {
		readCh := make(chan int64, 1)
		if !c.WithFlags(CLIENT_LUA) && c.MaxIdleTime == 0 {
			c.HeartBeatCh = make(chan int64, 1)
			// TODO 如果有问题，改成独立chan的
			go HeartBeating(s, c)
		}
		go ReadFromClient(s, c, readCh)
		select {
		case <-c.CloseCh:
			fmt.Println("ReadLoop ----> Stop Client")
			return
		case result := <-readCh:
			if result == C_OK {
				WriteToClient(s, c)
			}
		}
	}
}

func HeartBeating(s *Server, c *Client) {
	fmt.Println("HeartBeatLoop")
	s.wg.Add(1)
	defer s.wg.Done()
	select {
	case <-c.CloseCh:
		fmt.Println("HeartBeating ----> Close Client")
		close(c.HeartBeatCh)
		return
	case readCount := <-c.HeartBeatCh:
		fmt.Printf("HearBeat OK --> %d\n", readCount)
		return
	case <-time.After(c.MaxIdleTime):
		fmt.Println("HearBeat fail. 3s reached.")
		close(c.HeartBeatCh)
		BroadcastCloseClient(c)
		return
	}
}

func BroadcastCloseClient(c *Client) {
	close(c.CloseCh)
}

func BroadcastCloseServer(s *Server) {
	close(s.CloseCh)
}

func CloseClientListener(s *Server, c *Client) {
	s.wg.Add(1)
	defer s.wg.Done()
	select {
	case <-c.CloseCh:
		fmt.Println("CloseClientListener ----> Close Client")
		CloseClient(s, c)
		return
	}
}
