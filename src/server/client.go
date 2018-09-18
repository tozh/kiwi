package server

import (
	"time"
	"fmt"
	"sync"
	"bufio"
	"sync/atomic"
	"kiwi/src/evio"
	"net"
)

type Client struct {
	Id              int64
	Conn            evio.Conn
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
	QueryCount      int
	MaxIdleTime     time.Duration
	mutex           *sync.Mutex
}

func (c *Client) Context() interface{}       { return c.Conn.Context() }
func (c *Client) SetContext(ctx interface{}) { c.Conn.SetContext(ctx) }
func (c *Client) AddrIndex() int             { return c.Conn.AddrIndex() }
func (c *Client) LocalAddr() net.Addr        { return c.Conn.LocalAddr() }
func (c *Client) RemoteAddr() net.Addr       { return c.Conn.RemoteAddr() }


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
		c.PeerId = fmt.Sprintf("%kiwiS:0", s.UnixSocketPath)
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

func CatClientInfoString(c *Client) string {
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
	clientFmt := "id=%d addr=%kiwiS conn=%kiwiS name=%kiwiS age=%d idle=%d flags=%kiwiS db=%d cmd=%kiwiS"
	return fmt.Sprintf(clientFmt, c.Id, c.GetPeerId(kiwiS), c.Conn.LocalAddr().String(), c.Name, kiwiS.UnixTime.Sub(c.CreateTime).Nanoseconds()/1000,
		kiwiS.UnixTime.Sub(c.LastInteraction).Nanoseconds()/1000, flags.String(), c.Db.id, cmd)
}

func CreateClient(conn evio.Conn, flags int) *Client {
	createTime := kiwiS.UnixTime
	c := Client{
		Id:              0,
		Conn:            conn,
		Name:            "",
		ProcessBuf:      &LargeBuffer{},
		Cmd:             nil,
		CreateTime:      createTime,
		LastInteraction: createTime,
		Flags:           flags,
		Node:            nil,
		RequestType:     0,
		MultiBulkLen:    0,
		Authenticated:   0,
		QueryCount:      0,
		MaxIdleTime:     0,
		mutex:           &sync.Mutex{},
	}
	if !c.WithFlags(CLIENT_LUA) {
		c.MaxIdleTime = kiwiS.ClientMaxIdleTime
	}
	c.GetNextClientId(kiwiS)
	SelectDB(&c, 0)
	LinkClient(&c)
	return &c
}

func BroadcastCloseClient(c *Client) {
	close(c.CloseCh)
}

func CloseClientListener(c *Client) {
	kiwiS.wg.Add(1)
	defer kiwiS.wg.Done()
	select {
	case <-c.CloseCh:
		CloseClient(c)
	}
}

func HeartBeating(c *Client, readCh chan int) {
	// fmt.Println("HeartBeatLoop")
	kiwiS.wg.Add(1)
	defer kiwiS.wg.Done()
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

func LinkClient(c *Client) {
	kiwiS.Clients.ListAddNodeTail(c)
	kiwiS.ClientsMap[c.Id] = c
	c.Node = kiwiS.Clients.ListTail()
	atomic.AddInt64(&kiwiS.StatConnCount, 1)
}

func UnLinkClient(c *Client) {

	if c.Conn != nil {
		kiwiS.Clients.ListDelNode(c.Node)
		c.Node = nil
		delete(kiwiS.ClientsMap, c.Id)
		atomic.AddInt64(&kiwiS.StatConnCount, -1)
		c.Conn.Close()
		c.Conn = nil
	}
}

func CloseClient(c *Client) {
	// fmt.Println("CloseClient")
	c.ReadBuf = nil
	c.ReplyWriter = nil
	c.ResetArgv()
	c.ProcessBuf = nil
	c.ReplyWriter = nil
	UnLinkClient(c)
}

func SelectDB(c *Client, dbId int) int {
	if dbId < 0 || dbId >= kiwiS.DbNum {
		return C_ERR
	}
	c.Db = kiwiS.Dbs[dbId]
	return C_OK
}

func DbDeleteSync(c *Client, key string) bool {
	// TODO expire things
	c.Db.Delete(key)
	return true
}

func DbDeleteAsync(c *Client, key string) bool {
	// TODO
	c.Db.Delete(key)
	return true
}
