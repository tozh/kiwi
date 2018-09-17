package server

import (
	"net"
	"time"
	"fmt"
	"sync"
	"bufio"
	"sync/atomic"
)

type Client struct {
	Id              int64
	Conn            net.Conn
	Db              *Db
	Name            string
	ReadBuf         []byte
	ProcessBuf      *LargeBuffer
	Argc            int      // count of arguments
	Argv            []string // arguments of current command
	Cmd             *Command
	ReplyWriter     *bufio.Writer
	CreateTime      time.Time
	LastInteraction time.Time
	Flags           int
	Node            *ListNode
	PeerId          string
	RequestType     int // Request protocol type: PROTO_REQ_*
	MultiBulkLen    int // Number of multi bulk arguments left to read.
	Authenticated   int
	CloseCh         chan struct{}
	//HeartBeatCh     chan int
	QueryCount  int
	MaxIdleTime time.Duration
	mutex       *sync.Mutex
}

func (c *Client) GetLastInteraction() time.Time {
	return c.LastInteraction
}

func (c *Client) SetLastInteraction(time time.Time) {
	c.LastInteraction = time
}

func (c *Client) WithFlags(flags int) bool {
	return c.Flags&flags != 0
}

func (c *Client) AddFlags(flags int) {
	c.Flags |= flags
}

func (c *Client) DeleteFlags(flags int) {
	c.Flags &= ^flags
}

func (c *Client) GeneratePeerId(s *Server) {
	if c.WithFlags(CLIENT_UNIX_SOCKET) {
		c.PeerId = fmt.Sprintf("%s:0", s.UnixSocketPath)
	} else {
		c.PeerId = c.Conn.RemoteAddr().String()
	}
}

func (c *Client) GetPeerId(s *Server) string {
	if c.PeerId == "" {
		c.GeneratePeerId(s)
	}
	return c.PeerId
}

func (c *Client) GetNextClientId(s *Server) {
	c.Id = atomic.LoadInt64(&s.NextClientId)
	atomic.AddInt64(&s.NextClientId, 1)
}

func (c *Client) Write(b []byte) (int, error) {
	n, err := c.Conn.Write(b)
	return int(n), err
}

func (c *Client) Read(b []byte) (int, error) {
	n, err := c.Conn.Read(b)
	return int(n), err
}

func (c *Client) GetClientType() int {
	if c.WithFlags(CLIENT_MASTER) {
		return CLIENT_TYPE_MASTER
	}
	if c.WithFlags(CLIENT_SLAVE) && !c.WithFlags(CLIENT_MONITOR) {
		return CLIENT_TYPE_SLAVE
	}
	if c.WithFlags(CLIENT_TYPE_PUBSUB) {
		return CLIENT_TYPE_PUBSUB
	}
	return CLIENT_TYPE_NORMAL
}

func (c *Client) GetClientTypeByName(name string) int {
	switch name {
	case "normal":
		return CLIENT_TYPE_NORMAL
	case "slave":
		return CLIENT_TYPE_SLAVE
	case "pubsub":
		return CLIENT_TYPE_PUBSUB
	case "master":
		return CLIENT_TYPE_MASTER
	default:
		return -1
	}
}

func (c *Client) GetClientTypeName(ctype int) string {
	switch ctype {
	case CLIENT_TYPE_NORMAL:
		return "normal"
	case CLIENT_TYPE_SLAVE:
		return "slave"
	case CLIENT_TYPE_PUBSUB:
		return "pubsub"
	case CLIENT_TYPE_MASTER:
		return "master"
	default:
		return ""
	}
}


func (c *Client) ResetArgv() {
	c.Argc = 0
	c.Cmd = nil
	c.Argv = nil
}


func (c *Client) Reset() {
	c.ResetArgv()
	c.RequestType = 0
	c.MultiBulkLen = 0
	c.ReplyWriter.Reset(c.Conn)
	c.ProcessBuf.Reset()
}


func (c *Client) PrepareClientToWrite() int {
	// fmt.Println("PrepareClientToWrite")

	if c.WithFlags(CLIENT_REPLY_OFF | CLIENT_REPLY_SKIP) {
		// fmt.Println("PrepareClientToWrite111111")
		return C_ERR
	}
	if c.Conn == nil {
		// fmt.Println("PrepareClientToWrite222222")
		return C_ERR
	}
	// fmt.Println("PrepareClientToWrite-------OK")

	return C_OK
}

func CatClientInfoString(s *Server, c *Client) string {
	flags := Buffer{}
	if c.WithFlags(CLIENT_SLAVE) {
		if c.WithFlags(CLIENT_MONITOR) {
			flags.WriteByte('O')
		} else {
			flags.WriteByte('S')
		}
	}

	if c.WithFlags(CLIENT_MASTER) {
		flags.WriteByte('M')
	}
	if c.WithFlags(CLIENT_PUBSUB) {
		flags.WriteByte('P')
	}
	if c.WithFlags(CLIENT_MULTI) {
		flags.WriteByte('x')
	}
	if c.WithFlags(CLIENT_BLOCKED) {
		flags.WriteByte('b')
	}
	if c.WithFlags(CLIENT_DIRTY_CAS) {
		flags.WriteByte('d')
	}
	if c.WithFlags(CLIENT_CLOSE_AFTER_REPLY) {
		flags.WriteByte('c')
	}
	if c.WithFlags(CLIENT_UNBLOCKED) {
		flags.WriteByte('u')
	}
	if c.WithFlags(CLIENT_CLOSE_ASAP) {
		flags.WriteByte('A')
	}
	if c.WithFlags(CLIENT_UNIX_SOCKET) {
		flags.WriteByte('U')
	}
	if c.WithFlags(CLIENT_READONLY) {
		flags.WriteByte('r')
	}
	if flags.Len() == 0 {
		flags.WriteByte('N')
	}
	flags.WriteByte(0)
	flags.WriteByte('r')
	flags.WriteByte('w')
	flags.WriteByte(0)
	cmd := "nil"
	clientFmt := "id=%d addr=%s conn=%s name=%s age=%d idle=%d flags=%s db=%d cmd=%s"
	return fmt.Sprintf(clientFmt, c.Id, c.GetPeerId(s), c.Conn.LocalAddr().String(), c.Name, s.UnixTime.Sub(c.CreateTime).Nanoseconds()/1000,
		s.UnixTime.Sub(c.LastInteraction).Nanoseconds()/1000, flags.String(), c.Db.id, cmd)
}

func CreateClient(s *Server, conn net.Conn, flags int) *Client {
	createTime := s.UnixTime
	c := Client{
		Id:              0,
		Conn:            conn,
		Name:            "",
		ReadBuf:         make([]byte, 0),
		ProcessBuf:      &LargeBuffer{},
		Argc:            0,                 // count of arguments
		Argv:            make([]string, 0), // arguments of current command
		Cmd:             nil,
		ReplyWriter:     bufio.NewWriter(conn),
		CreateTime:      createTime,
		LastInteraction: createTime,
		Flags:           flags,
		Node:            nil,
		PeerId:          "",
		RequestType:     0,
		MultiBulkLen:    0,
		Authenticated:   0,
		CloseCh:         make(chan struct{}, 1),
		//HeartBeatCh:     nil,
		QueryCount:  0,
		MaxIdleTime: 0,
		mutex:       &sync.Mutex{},
	}
	if !c.WithFlags(CLIENT_LUA) {
		c.MaxIdleTime = s.ClientMaxIdleTime
	}
	c.GetNextClientId(s)
	SelectDB(s, &c, 0)
	LinkClient(s, &c)
	return &c
}

func BroadcastCloseClient(c *Client) {
	close(c.CloseCh)
}

func CloseClientListener(s *Server, c *Client) {
	s.wg.Add(1)
	defer s.wg.Done()
	select {
	case <-c.CloseCh:
		CloseClient(s, c)
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


func LinkClient(s *Server, c *Client) {
	s.Clients.ListAddNodeTail(c)
	s.ClientsMap[c.Id] = c
	c.Node = s.Clients.ListTail()
	atomic.AddInt64(&s.StatConnCount, 1)
}

func UnLinkClient(s *Server, c *Client) {

	if c.Conn != nil {
		s.Clients.ListDelNode(c.Node)
		c.Node = nil
		delete(s.ClientsMap, c.Id)
		atomic.AddInt64(&s.StatConnCount, -1)
		c.Conn.Close()
		c.Conn = nil
	}
}

func CloseClient(s *Server, c *Client) {
	// fmt.Println("CloseClient")
	c.ReadBuf = nil
	c.ReplyWriter = nil
	c.ResetArgv()
	c.ProcessBuf = nil
	c.ReplyWriter = nil
	UnLinkClient(s, c)
}

func SelectDB(s *Server, c *Client, dbId int) int {
	if dbId < 0 || dbId >= s.DbNum {
		return C_ERR
	}
	c.Db = s.Dbs[dbId]
	return C_OK
}

func DbDeleteSync(s *Server, c *Client, key string) bool {
	// TODO expire things
	c.Db.Delete(key)
	return true
}

func DbDeleteAsync(s *Server, c *Client, key string) bool {
	// TODO
	c.Db.Delete(key)
	return true
}