package server

import (
	. "redigo/src/structure"
	. "redigo/src/db"
	"bytes"
	"time"
	"sync"
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
	Buf bytes.Buffer
	BufPos int64
	SentLen int64
	Flags int64
}

func (s *Server) CreateClient() *Client {
	c := Client{

	}
}



