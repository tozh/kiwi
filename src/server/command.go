package server

import (
	. "redigo/src/object"
	. "redigo/src/constant"
	"strings"
	"strconv"
	"time"
)

type CommandProcess func(s *Server, c *Client)

type Command struct {
	Name  string
	Arity int64
	Flags int64
	/* What keys should be loaded in background when calling this command? */
	FirstKey int64
	LastKey  int64
	KeyStep  int64
	Duration time.Duration
	Calls    int64
	Process  CommandProcess
}

func (cmd *Command) WithFlags(flags int64) bool {
	return cmd.Flags&flags != 0
}

func (cmd *Command) AddFlags(flags int64) {
	cmd.Flags |= flags
}

func (cmd *Command) DeleteFlags(flags int64) {
	cmd.Flags &= ^flags
}

func (cmd *Command) IsProcess(cmdP *CommandProcess) bool {
	return &cmd.Process == cmdP
}

/* SET key value [NX] [XX] [EX <seconds>] [PX <milliseconds>] */
// NX - not exist
// XX - exist
// EX - expire in seconds
// PX - expire in milliseconds
func SetGenericCommand(s *Server, c *Client, flags int64, key string, ok_reply string, abort_reply string) {
	if (flags&OBJ_SET_NX != 0 && c.Db.Exist(key)) || (flags&OBJ_SET_XX != 0 && !c.Db.Exist(key)) {
		if abort_reply != "" {
			s.AddReply(c, s.Shared.NullBulk)
		} else {
			s.AddReply(c, abort_reply)
		}
		return
	}
	o := CreateStrObjectByStr(s, c.Argv[2])
	c.Db.Set(key, o)
	s.Dirty++
	if ok_reply != "" {
		s.AddReply(c, s.Shared.Ok)
	} else {
		s.AddReply(c, ok_reply)
	}
}

var SetCommand CommandProcess = func(s *Server, c *Client) {
	flags := OBJ_SET_NO_FLAGS
	for j := int64(3); j < c.Argc; j++ {
		a := strings.ToUpper(c.Argv[j])

		if a == "NX" && (flags&OBJ_SET_XX) != 0 {
			flags |= OBJ_SET_NX
		} else if a == "XX" && (flags&OBJ_SET_NX) != 0 {
			flags |= OBJ_SET_XX
		} else {
			s.AddReply(c, s.Shared.SyntaxErr)
			return
		}
		// TODO expire is not implemented now, so end here
	}

	SetGenericCommand(s, c, int64(flags), c.Argv[1], "", "")
}

var SetNxCommand CommandProcess = func(s *Server, c *Client) {
	SetGenericCommand(s, c, OBJ_SET_NX, c.Argv[1], s.Shared.One, s.Shared.Zero)
}

var SetExCommand CommandProcess = func(s *Server, c *Client) {
	SetGenericCommand(s, c, OBJ_SET_XX, c.Argv[1], "", "")
}

var FlushAllCommand CommandProcess = func(s *Server, c *Client) {
	if s.ConfigFlushAll {
		c.Db.FlushAll()
		s.AddReply(c, s.Shared.Ok)
	}
	s.Dirty++
}

//func (s *Server) ExistCommand(c *Client) {
//	c.Db.Exist(c.Argv[1])
//	// expire
//	// addReply()
//}

func IncrDecrCommand(s *Server, c *Client, incr int64) {
	o := c.Db.Get(c.Argv[1]).(*StrObject)
	if o == nil {
		o = CreateStrObjectByInt(s, incr)
		c.Db.Set(c.Argv[1], o)
	}
	if !IsStrObjectInt(o) {
		return
	}
	value := *o.Value.(*int64)
	oldValue := value
	value += incr
	if IsOverflowInt(oldValue, incr) {
		s.AddReplyError(c, "increment or decrement would overflow")
		return
	}
	ReplaceStrObjectByInt(s, o, &oldValue, &value)
	s.Dirty++
	s.AddReplyInt(c, value)
}

var IncrCommand CommandProcess = func(s *Server, c *Client) {
	IncrDecrCommand(s, c, 1)
}

var DecrCommand CommandProcess = func(s *Server, c *Client) {
	IncrDecrCommand(s, c, -1)
}

var IncrByCommand CommandProcess = func(s *Server, c *Client) {
	incr, err := strconv.ParseInt(c.Argv[2], 10, 64)
	if err != nil {
		return
	}
	IncrDecrCommand(s, c, incr)
}

var DecrByCommand CommandProcess = func(s *Server, c *Client) {
	decr, err := strconv.ParseInt(c.Argv[2], 10, 64)
	if err != nil {
		return
	}
	IncrDecrCommand(s, c, -decr)
}

var StrLenCommand CommandProcess = func(s *Server, c *Client) {
	o := DbGetOrReply(s, c, c.Argv[1], s.Shared.NullBulk)
	if o == nil {
		return
	}
	str, err := GetStrObjectValueString(o.(*StrObject))
	if err == nil {
		return
	}
	s.AddReplyInt(c, int64(len(str)))
}

// Cat strings
var AppendCommand CommandProcess = func(s *Server, c *Client) {
	var length int64
	o := c.Db.Get(c.Argv[1]).(*StrObject)
	if o == nil {
		o = CreateStrObjectByStr(s, c.Argv[2])
		c.Db.Set(c.Argv[1], o)
	} else {
		if !CheckRType(o, OBJ_RTYPE_STR) {
			return
		}
		_, length = AppendStrObject(s, o, c.Argv[2])
	}
	s.Dirty++
	s.AddReplyInt(c, length)
}

func DbGetOrReply(s *Server, c *Client, key string, reply string) IObject {
	o := c.Db.Get(key)
	if o == nil {
		s.AddReply(c, reply)
	}
	return o
}

func GetGenericCommand(s *Server, c *Client) int64 {
	o := DbGetOrReply(s, c, c.Argv[1], s.Shared.NullBulk)
	if o == nil {
		return C_OK
	}
	if !CheckRType(o, OBJ_RTYPE_STR) {
		s.AddReply(c, s.Shared.WrongTypeErr)
		return C_ERR
	} else {
		s.AddReplyStrObj(c, o.(*StrObject))
		return C_OK
	}
}

var GetCommand CommandProcess = func(s *Server, c *Client) {
	GetGenericCommand(s, c)
}

func GetSetCommand(s *Server, c *Client) {
	if GetGenericCommand(s, c) == C_ERR {
		return
	}
	o := CreateStrObjectByStr(s, c.Argv[2])
	c.Db.Set(c.Argv[1], o)
	s.Dirty++
}

func MSetGenericCommand(s *Server, c *Client, flags int64) {
	if c.Argc%2 == 0 {
		s.AddReplyError(c, "wrong number of arguments for MSET")
		return
	}
	// check the nx flag
	// The MSETNX sematic is to return zero and set nothing if one of keys exists
	existKeyCount := 0
	if flags&OBJ_SET_NX != 0 {
		for j := 1; j < len(c.Argv); j += 2 {
			if c.Db.Exist(c.Argv[j]) {
				existKeyCount++
			}
		}
		if existKeyCount != 0 {
			// addReply(c, s.Shared.CommandZero)
			return
		}
	}
	for j := 1; j < len(c.Argv); j += 2 {
		o := CreateStrObjectByStr(s, c.Argv[j+1])
		c.Db.Set(c.Argv[j], o)
	}
	s.Dirty += (c.Argc - 1) / 2
	if flags&OBJ_SET_NX != 0 {
		s.AddReply(c, s.Shared.One)
	} else {
		s.AddReply(c, s.Shared.Ok)
	}
}

var MSetCommand CommandProcess = func(s *Server, c *Client) {
	MSetGenericCommand(s, c, OBJ_SET_NO_FLAGS)
}

var MSetNxCommand CommandProcess = func(s *Server, c *Client) {
	MSetGenericCommand(s, c, OBJ_SET_NX)
}

var MGetCommand CommandProcess = func(s *Server, c *Client) {
	s.AddReplyMultiBulkLength(c, c.Argc-1)
	for j := 1; j < len(c.Argv); j++ {
		o := c.Db.Get(c.Argv[j]).(*StrObject)
		if o == nil {
			s.AddReply(c, s.Shared.NullBulk)
		} else {
			if !CheckRType(o, OBJ_RTYPE_STR) {
				s.AddReply(c, s.Shared.NullBulk)
			} else {
				s.AddReplyBulk(c, o)
			}
		}
	}
}

var AuthCommand CommandProcess = func(s *Server, c *Client) {
	if s.RequirePassword == nil {
		s.AddReplyError(c, "Client sent AUTH, but no password is set")
	} else if c.Argv[1] == *s.RequirePassword {
		c.Authenticated = 1
		s.AddReply(c, s.Shared.Ok)
	} else {
		c.Authenticated = 0
		s.AddReplyError(c, "invalid password")
	}
}

//var ExecCommand CommandProcess = func(s *Server, c *Client) {
//	// TODO:finish this function
//	return
//}
//
//var MultiCommand CommandProcess = func(s *Server, c *Client) {
//	if c.WithFlags(CLIENT_MULTI) {
//		s.AddReplyError(c, "MULTI calls can not be nested")
//		return
//	}
//	c.AddFlags(CLIENT_MULTI)
//	s.AddReply(c, s.Shared.Ok)
//}
//
//var DiscardCommand CommandProcess = func(s *Server, c *Client) {
//	if c.WithFlags(CLIENT_MULTI) {
//		s.AddReplyError(c, "DISCARD without MULTI")
//		return
//	}
//}
