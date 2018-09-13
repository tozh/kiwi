package test_server

import (
	"net"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"sync"
	"io"
	"time"
)

type accepted struct {
	conn net.Conn
	err  error
}

type Server struct {
	CloseCh        chan struct{}
	UnixSocketPath string
	Port           int64
	Ip             string
	wg             sync.WaitGroup
	count          int
}

type Client struct {
	CloseCh     chan struct{}
	HeartBeatCh chan struct{}
	Conn        net.Conn
	ReadBuf     []byte
	WriteBuf    []byte
	mutex       sync.RWMutex
}

//func UnixServer(s *Server) {
//	s.wg.Add(1)
//	defer s.wg.Done()
//	fmt.Println(s.UnixSocketPath)
//	fmt.Println("------>UnixServer")
//	listener := AnetListenUnix(s.UnixSocketPath)
//	if listener == nil {
//		return
//	}
//	for {
//		ch := make(chan accepted, 1)
//		go func() {
//			conn, err := listener.Accept()
//			ch <- accepted{conn, err}
//		}()
//		select {
//		case <-s.CloseCh:
//			listener.Close()
//			return
//		case acc := <-ch:
//			if acc.err != nil {
//				continue
//			}
//			fmt.Printf("-->%v\n", "UnixServer ------ CommonServer")
//			CommonServer(s, acc.conn)
//		}
//	}
//}

func TcpServer(s *Server) {
	s.wg.Add(1)
	s.count++
	defer s.wg.Done()
	defer func() {
		s.count--
	}()
	fmt.Println("------>TcpServer")
	listener := AnetListenTcp("tcp", s.Ip, s.Port)
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
			listener.Close()
			return
		case acc := <-ch:
			if acc.err != nil {
				continue
			}
			fmt.Printf("-->%v\n", "TcpServer ------ CommonServer")
			CommonServer(s, acc.conn)
		}
	}
}

func CommonServer(s *Server, conn net.Conn) {
	c := CreateClient(conn)
	go ReadLoop(s, c)
	go ClientCloseListener(s, c)
}

func ReadLoop(s *Server, c *Client) {

	fmt.Println("ReadLoop")
	s.wg.Add(1)
	s.count++
	defer s.wg.Done()
	defer func() {
		s.count--
	}()
	for {
		select {
		case <-s.CloseCh:
			fmt.Println("ReadLoop ----> Stop Server")
			return
		case <-c.CloseCh:
			fmt.Println("ReadLoop ----> Stop Client")
			return
		default:
			//c.HeartBeatCh = make(chan struct{}, 1)
			ProcessClient(c)
		}
	}
}
//
//func WriteLoop(s *Server, c *Client) {
//	fmt.Println("WriteLoop")
//	s.wg.Add(1)
//	s.count++
//	defer s.wg.Done()
//	defer func() {
//		s.count--
//	}()
//	for {
//		select {
//		case <-s.CloseCh:
//			fmt.Println("WriteLoop ----> Stop Server")
//			return
//		case <-c.CloseCh:
//			fmt.Println("WriteLoop ----> Stop Client")
//			return
//		case <-c.WriteCh:
//			go
//			return
//		}
//	}
//}

func ClientCloseListener(s *Server, c *Client) {
	s.wg.Add(1)
	defer s.wg.Done()

	select {
	case <-c.CloseCh:
		CloseClient(c)
		return
	}
}

func HeartBeating(s *Server, c *Client) {
	fmt.Println("HeartBeatLoop")
	s.wg.Add(1)
	defer s.wg.Done()

	select {
	case <-c.CloseCh:
		fmt.Println("HeartBeating ----> Stop Client")
		return
	case <-c.HeartBeatCh:
		fmt.Println("HearBeat OK")
		return
	case <-time.After(time.Second * 3):
		fmt.Println("HearBeatFail, 3s reached!!!")
		close(c.CloseCh)
		return
	}
}

func ProcessClient(c *Client) {
	fmt.Println("-->ProcessClient")
	n, err := c.Conn.Read(c.ReadBuf)
	if err != nil {
		fmt.Println("ProcessClient:", err)
		if err == io.EOF {
			fmt.Println("ProcessClient: EOF !!!!")
			close(c.CloseCh)
		}
		return
	}
	//c.HeartBeatCh <- struct{}{}
	recieved := c.ReadBuf[:n]
	fmt.Println("Server Recieved:", string(recieved))
	c.WriteBuf = append([]byte("----->"), recieved...)
	WriteToClient(c)
}

func WriteToClient(c *Client) {
	fmt.Println("-->WriteToClient")
	fmt.Println("Server Send:", string(c.WriteBuf))
	_, err := c.Conn.Write(c.WriteBuf)
	if err != nil {
		fmt.Println("WriteToClient:", err)
		//close(c.CloseCh)
	}
}

func CreateClient(conn net.Conn) *Client {
	fmt.Println("CreateClient")
	return &Client{
		make(chan struct{}, 1),
		make(chan struct{}, 1),
		conn,
		make([]byte, 4),
		make([]byte, 4),
		sync.RWMutex{},
	}
}

func CloseClient(c *Client) {
	c.Conn.Close()
}


func CreateServer() *Server {
	return &Server{
		make(chan struct{}),
		os.TempDir() + "my_test_server.sock",
		6699,
		"0.0.0.0",
		sync.WaitGroup{},
		0,
	}
}

func StartServer(s *Server) {
	go TcpServer(s)
	//go UnixServer(s)
}

func StopServer(s *Server) {
	close(s.CloseCh)
}

func HandleSignal(s *Server) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	<-c
	StopServer(s)
	fmt.Println("s.count----->", s.count)
	s.wg.Wait()
}
