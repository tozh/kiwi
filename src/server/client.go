package server

import (
	. "redigo/src/structure"
	. "redigo/src/db"
	. "redigo/src/constant"
)

type Client struct {
	Id int64
	Fd int64
	Db *Db
	Name string
	QueryBuf string // buffer use to accumulate client query
	QueryBufPeak int64
	Argc int64       // count of arguments
	Argv []string // arguments of current command
	Cmd *RedisCommand
	LastCmd *RedisCommand
	Reply *List
	ReplySize int64
	SentSize int64 // Amount of bytes already sent in the current buffer or object being sent.
	CreateTime int64
	LastInteraction  int64
	Buf []byte
	BufPos int64
	SentLen int64
	Flags int64
}

func (c *Client) WithFlags(flags int64) bool {
	return c.Flags & flags != 0
}

func (c *Client) AddFlags(flags int64) {
	c.Flags |= flags
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
	if c.WithFlags(CLIENT_CLOSE_AFTER_REPLY){
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

func (c *Client) AddReplyStringToList(str string) {
	if c.WithFlags(CLIENT_CLOSE_AFTER_REPLY) {
		return
	}
	c.Reply.ListAddNodeTail(&str)
	c.ReplySize += int64(len(str))
}

// functions for client
func CopyClientOutputBuffer(dst *Client, src *Client) {
	dst.Reply.ListEmpty()
	dst.Reply = ListDup(src.Reply)
	copy(dst.Buf, src.Buf[0:src.BufPos])
	dst.BufPos = src.BufPos
	dst.ReplySize = src.ReplySize
}

