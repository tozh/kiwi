package server

import (
	. "redigo/src/structure"
	. "redigo/src/db"
	. "redigo/src/constant"
	"net"
	"time"
	"bytes"
)

type Client struct {
	Id               int64
	Conn             net.Conn
	Db               *Db
	Name             string
	QueryBuf         []byte // buffer use to accumulate client query
	QueryBufPeak     int64
	Argc             int64    // count of arguments
	Argv             []string // arguments of current command
	Cmd              *RedisCommand
	LastCmd          *RedisCommand
	Reply            *List
	ReplySize        int64
	SentSize         int64 // Amount of bytes already sent in the current buffer or object being sent.
	CreateTime       time.Duration
	LastInteraction  time.Duration
	Buf              []byte
	BufPos           int64
	SentLen          int64
	Flags            int64
	Node             *ListNode
	PendingWriteNode *ListNode
	UnblockedNode    *ListNode
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

func (c *Client) SelectDB(s *Server, dbId int64) int64 {
	if dbId < 0 || dbId >= s.DbNum {
		return C_ERR
	}
	c.Db = s.Dbs[dbId]
	return C_OK
}

func (c *Client) GetNextClientId(s *Server) {
	s.mutex.Lock()
	c.Id = s.NextClientId
	s.NextClientId++
	s.mutex.Unlock()
}

func (c *Client) HasPendingReplies() bool {
	return c.BufPos != 0 || c.Reply.ListLength() != 0
}

func (c *Client) AddReplyToBuffer(str string) int64 {
	if c.WithFlags(CLIENT_CLOSE_AFTER_REPLY) {
		return C_OK
	}
	if c.Reply.ListLength() > 0 {
		return C_ERR
	}
	available := cap(c.Buf)
	if len(str) > available {
		return C_ERR
	}
	copy(c.Buf[c.BufPos:], str)
	c.BufPos += int64(len(str))
	return C_OK
}

func (c *Client) Write(b []byte) (int64, error) {
	n, err := c.Conn.Write(b)
	return int64(n), err
}

func (c *Client) Read(b []byte) (int64, error) {
	n, err := c.Conn.Read(b)
	return int64(n), err
}

func (c *Client) AddReplyStringToList(str string) {
	if c.WithFlags(CLIENT_CLOSE_AFTER_REPLY) {
		return
	}
	c.Reply.ListAddNodeTail(&str)
	c.ReplySize += int64(len(str))
}

func (c *Client) CatClientInfoString() {
	flags := bytes.Buffer{}
	if c.WithFlags(CLIENT_SLAVE) {
		if c.WithFlags(CLIENT_MONITOR) {
			flags.WriteByte('O')
		} else {
			flags.WriteByte('S')
		}
	}


	if c.WithFlags(CLIENT_MASTER) { flags.WriteByte('M') }
	if c.WithFlags(CLIENT_PUBSUB) { flags.WriteByte('P') }
	if c.WithFlags(CLIENT_MULTI) { flags.WriteByte('x') }
	if c.WithFlags(CLIENT_BLOCKED) { flags.WriteByte('b') }
	if c.WithFlags(CLIENT_DIRTY_CAS) { flags.WriteByte('d') }
	if c.WithFlags(CLIENT_CLOSE_AFTER_REPLY) { flags.WriteByte('c') }
	if c.WithFlags(CLIENT_UNBLOCKED) { flags.WriteByte('u') }
	if c.WithFlags(CLIENT_CLOSE_ASAP) { flags.WriteByte('A') }
	if c.WithFlags(CLIENT_UNIX_SOCKET) { flags.WriteByte('U') }
	if c.WithFlags(CLIENT_READONLY) { flags.WriteByte('r') }
	if flags.Len() == 0 {
		flags.WriteByte('N')
	}
	flags.WriteByte(0)


}

func (c *Client) GetAllClientInfoString() {

}

func (c *Client) ClientCommand() {

}

func (c *Client) GetClientType() {

}

func (c *Client) GetClientOutputBufferMemoryUsage() {

}

func (c *Client) CheckClientOutputBufferLimits() {

}

// resetClient prepare the client to process the next command
func (c *Client) Reset() {

}

// functions for client
func CopyClientOutputBuffer(dst *Client, src *Client) {
	dst.Reply.ListEmpty()
	dst.Reply = ListDup(src.Reply)
	copy(dst.Buf, src.Buf[0:src.BufPos])
	dst.BufPos = src.BufPos
	dst.ReplySize = src.ReplySize
}

