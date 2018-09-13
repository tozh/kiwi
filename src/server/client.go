package server

import (
	. "redigo/src/structure"
	. "redigo/src/constant"
	"net"
	"time"
	"fmt"
	"unsafe"
	"bytes"
	"sync"
)

type Client struct {
	Id              int64
	Conn            net.Conn
	Db              *Db
	Name            string
	QueryBuf        []byte // buffer use to accumulate client query
	QueryBufSize	int64
	QueryBufPeak    int64
	Argc            int64    // count of arguments
	Argv            []string // arguments of current command
	Cmd             *Command
	LastCmd         *Command
	Reply           *List
	ReplySize       int64
	CreateTime      time.Time
	LastInteraction time.Time
	Buf             []byte
	BufPos          int64
	SentLen         int64
	Flags           int64
	Node            *ListNode
	PeerId          string
	RequestType     int64 // Request protocol type: PROTO_REQ_*
	MultiBulkLen    int64 // Number of multi bulk arguments left to read.
	BulkLen         int64 // Length of bulk argument in multi bulk request.
	Authenticated   int64
	CloseCh         chan struct{}
	HeartBeatCh     chan int64
	ReadCount int64
	mutex           sync.RWMutex
}

func (c *Client) GetLastInteraction() time.Time {
	return c.LastInteraction
}

func (c *Client) SetLastInteraction(time time.Time) {
	c.LastInteraction = time
}

func (c *Client) WithFlags(flags int64) bool {
	return c.Flags&flags != 0
}

func (c *Client) AddFlags(flags int64) {
	c.Flags |= flags
}

func (c *Client) DeleteFlags(flags int64) {
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
	s.mutex.Lock()
	defer s.mutex.Unlock()
	c.Id = s.NextClientId
	s.NextClientId++
}

func (c *Client) HasPendingReplies() bool {
	return c.BufPos != 0 || c.Reply.ListLength() != 0
}

func (c *Client) Write(b []byte) (int64, error) {
	n, err := c.Conn.Write(b)
	return int64(n), err
}

func (c *Client) Read(b []byte) (int64, error) {
	n, err := c.Conn.Read(b)
	return int64(n), err
}

func (c *Client) GetClientType() int64 {
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

func (c *Client) GetClientTypeByName(name string) int64 {
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

func (c *Client) GetClientTypeName(ctype int64) string {
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

func (c *Client) GetOutputBufferMemoryUsage() int64 {
	listNodeSize := int64(unsafe.Sizeof(ListNode{}))
	listSize := int64(unsafe.Sizeof(List{}))
	return c.ReplySize + c.Reply.ListLength()*listNodeSize + listSize
}

// resetClient prepare the client to process the next command
func (c *Client) Reset() {
	c.ResetArgv()
	c.RequestType = 0
	c.MultiBulkLen = 0
	c.BulkLen = 1

	c.DeleteFlags(CLIENT_REPLY_SKIP)
	if c.WithFlags(CLIENT_REPLY_SKIP_NEXT) {
		c.AddFlags(CLIENT_REPLY_SKIP)
		c.DeleteFlags(CLIENT_REPLY_SKIP_NEXT)
	}
}

func (c *Client) ResetArgv() {
	c.Argc = 0
	c.Cmd = nil
	c.Argv = nil
}

func (c *Client) PrepareClientToWrite() int64 {
	if c.WithFlags(CLIENT_REPLY_OFF | CLIENT_REPLY_SKIP) {
		return C_ERR
	}
	if c.Conn == nil {
		return C_ERR
	}
	return C_OK
}

// functions for client
func CopyClientOutputBuffer(dst *Client, src *Client) {
	dst.Reply.ListEmpty()
	dst.Reply = DupList(src.Reply)
	copy(dst.Buf, src.Buf[0:src.BufPos])
	dst.BufPos = src.BufPos
	dst.ReplySize = src.ReplySize
}

func CatClientInfoString(s *Server, c *Client) string {
	flags := bytes.Buffer{}
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
	if c.Cmd != nil {
		cmd = c.LastCmd.Name
	}

	clientFmt := "id=%d addr=%s conn=%s name=%s age=%d idle=%d flags=%s db=%d cmd=%s"
	return fmt.Sprintf(clientFmt, c.Id, c.GetPeerId(s), c.Conn.LocalAddr().String(), c.Name, s.UnixTime.Sub(c.CreateTime).Nanoseconds()/1000,
		s.UnixTime.Sub(c.LastInteraction).Nanoseconds()/1000, flags.String(), c.Db.Id, cmd)
}



