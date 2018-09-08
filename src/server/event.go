package server

import (
	"net"
	. "redigo/src/networking"
	. "redigo/src/constant"
)

type Dual struct {
	S *Server
	C *Client
}
type Chans struct {
	ReadCh  chan *Dual
	WriteCh chan *Dual
	CloseCh chan *Dual
	ErrCh   chan string
}

func UnixServer(s *Server, chs Chans) {
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
		CommonServer(s, conn, 0, "")

	}
}

func TcpServer(s *Server, addr string, flags int64) {
	listener := AnetListenTcp("tcp", addr, s.Port)
	if listener == nil {
		return
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			AnetSetErrorFormat("Tcp Accept error: %s", err)
			continue
		}
		CommonServer(s, conn, 0, addr)
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

	chs := Chans{
		make(chan *Dual),
		make(chan *Dual),
		make(chan *Dual),
		make(chan string),
	}
	dual := &Dual{s, c}
	chs.ReadCh <- dual
	go ReadLoop(chs, dual.S.CloseCh)
	go WriteLoop(chs, dual.S.CloseCh)
	go CloseLoop(chs, dual.S.CloseCh)
}

func ReadLoop(chs Chans, sCloseCh chan struct{}) {
	var dual *Dual
	for {
		select {
		case <-chs.CloseCh:
			return
		case <-sCloseCh:
			return
		case dual = <-chs.ReadCh:
			ReadQueryFromClient(dual)
			chs.WriteCh <- dual
		}
	}
}

func WriteLoop(chs Chans, sCloseCh chan struct{}) {
	var dual *Dual
	for {
		select {
		case <-chs.CloseCh:
			return
		case <-sCloseCh:
			return
		case dual = <-chs.WriteCh:
			WriteToClient(dual)
			chs.CloseCh <- dual
		}
	}
}

func CloseLoop(chs Chans, sCloseCh chan struct{}) {
	var dual *Dual
	for {
		select {
		case sCloseCh:
			return
		case dual = <-chs.CloseCh:
			if dual.C.WithFlags(CLIENT_CLOSE_AFTER_REPLY) {
				CloseClient(dual.S, dual.C)
			} else {
				CloseClientAsync(dual.S, dual.C)
			}
		}
	}
}

func TimeCron() {

}

