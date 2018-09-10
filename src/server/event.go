package server

import (
	"net"
	. "redigo/src/networking"
	. "redigo/src/constant"
	"fmt"
)

func UnixServer(s *Server) {
	listener := AnetListenUnix(s.UnixSocketPath)
	if listener == nil {
		return
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			AnetSetErrorFormat("Tcp Accept error: %s", err)
			continue
		}
		CommonServer(s, conn, CLIENT_UNIX_SOCKET, "")

	}
}

func TcpServer(s *Server, ip string) {
	fmt.Println("------>TcpServer")
	listener := AnetListenTcp("tcp", ip, s.Port)
	if listener == nil {
		return
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			AnetSetErrorFormat("Tcp Accept error: %s", err)
			continue
		}
		CommonServer(s, conn, 0, ip)
	}
}

func CommonServer(s *Server, conn net.Conn, flags int64, ip string) {
	c := s.CreateClient(conn)
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

1) Just disable protected mode sending the command 'CONFIG SET protected-mode no' from the loopback interface by connecting to Redis from the same host the server is running, however MAKE SURE Redis is not publicly accessible from internet if you do so. Use CONFIG REWRITE to make this change permanent.
2) Alternatively you can just disable the protected mode by editing the Redis configuration file, and setting the protectedmode option to 'no', and then restarting the server.
3) If you started the server manually just for testing, restart it with the '--protected-mode no' option.
4) Setup a bind address or an authentication password. 

NOTE: You only need to do one of the above things in order for the server to start accepting connections from the outside.\r\n`)
		conn.Write(err)
		s.StatRejectedConn++
	}
	c.AddFlags(flags)

	c.ReadCh <- struct {}{}
	go ReadLoop(s, c)
	go WriteLoop(s, c)
	go CloseLoop(s, c)
}

func ReadLoop(s* Server, c* Client) {
	fmt.Println("ReadLoop")
	for {
		select {
		case <-c.CloseCh:
			fmt.Println("ReadLoop ----> Stop Client")
			return

		case <-c.ReadCh:
			ReadQueryFromClient(s, c)
			c.WriteCh <- struct{}{}
		}
	}
}

func WriteLoop(s* Server, c* Client) {
	fmt.Println("WriteLoop")
	for {
		select {
		case <-c.CloseCh:
			fmt.Println("WriteLoop ----> Stop Client")
			return

		case <-c.WriteCh:
			WriteToClient(s, c)
			c.CloseCh <- struct {}{}
		}
	}
}

func CloseLoop(s* Server, c* Client) {
	fmt.Println("CloseLoop")
	for {
		select {
		case <-c.CloseCh:
			if c.WithFlags(CLIENT_CLOSE_AFTER_REPLY) {
				fmt.Println("CloseLoop ----> Stop Client Sync")
				CloseClient(s, c)
			} else {
				fmt.Println("CloseLoop ----> Stop Client Async")
				CloseClientAsync(s, c)
			}
		}
	}
}

func TimeCron() {

}

