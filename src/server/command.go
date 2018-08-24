package server

import (
	."redigo/src/object"
	."redigo/src/constant"
	"strings"
	"strconv"
)

/* SET key value [NX] [XX] [EX <seconds>] [PX <milliseconds>] */
// NX - not exist
// XX - exist
// EX - expire in seconds
// PX - expire in milliseconds

func (s *Server) SetGenericCommand(c *Client, flags int64, key string) {
	if (flags & OBJ_SET_NX != 0 && c.Db.Exist(key)) ||
		(flags & OBJ_SET_XX != 0 && !c.Db.Exist(key)) {
		// addReply
		return
	}
	o := CreateStrObjectByStr(s, c.Argv[2])
	c.Db.Set(key, o)
	s.Dirty++

	//if expire

	// addReply
}

func (s *Server) SetCommand(c *Client) {
	flags := OBJ_SET_NO_FLAGS
	for j:=int64(3); j<c.Argc;j++ {
		a := strings.ToUpper(c.Argv[j])

		if a == "NX" && (flags & OBJ_SET_XX) != 0 {
			flags |= OBJ_SET_NX
		} else if a == "XX" && (flags & OBJ_SET_NX) != 0 {
			flags |= OBJ_SET_XX
		} else {
			// addReply
			return
		}
		// expire is not implemented now, so end here
	}

	s.SetGenericCommand(c, int64(flags), c.Argv[1])
}

func (s *Server) SetNxCommand(c *Client) {
	s.SetGenericCommand(c, OBJ_SET_NX, c.Argv[1])
}

func (s *Server) FlushAllCommand(c *Client) {
	if s.ConfigFlushAll {
		c.Db.FlushAll()
		// addReply
	} else {
		// addReplyError(c, "cannot flush all")
	}
}

func (s *Server) ExistCommand(c *Client) {
	c.Db.Exist(c.Argv[1])
	// expire
	// addReply()
}

func (s *Server) IncrDecrCommand(c *Client, incr int64) {
	o := c.Db.Get(c.Argv[1]).(*StrObject)
	if o == nil {
		o = CreateStrObjectByInt64(s, incr)
		c.Db.Set(c.Argv[1], o)
	}
	if !IsStrObjectInt64(o) {
		return
	}
	value := *o.Value.(*int64)
	oldValue := value
	value += incr
	if IsOverflowInt64(oldValue, incr) {
		// addReplyError(c, "increment or decrement would overflow")
		return
	}
	ReplaceStrObjectByInt64(s, o, &oldValue, &value)
}

func (s *Server) IncrCommand(c *Client) {
	s.IncrDecrCommand(c, 1)
}

func (s *Server) DecrCommand(c *Client) {
	s.IncrDecrCommand(c, -1)
}

func (s *Server) IncrByCommand(c *Client) {
	incr, err := strconv.ParseInt(c.Argv[2], 10, 64)
	if err != nil {
		return
	}
	s.IncrDecrCommand(c, incr)
}

func (s *Server) DecrByCommand(c *Client) {
	decr, err := strconv.ParseInt(c.Argv[2], 10, 64)
	if err != nil {
		return
	}
	s.IncrDecrCommand(c, -decr)
}

func (s *Server) StrLenCommand(c *Client) {
	o := c.Db.Get(c.Argv[1]).(*StrObject)
	if o == nil {
		return
	}
	//str, err := GetStrObjectValueString(o)
	_, err := GetStrObjectValueString(o)
	if err == nil {
		return
	}
	// addReply(len(str))
}

// Cat strings
func (s *Server) AppendCommand(c *Client) {
	o := c.Db.Get(c.Argv[1]).(*StrObject)
	if o == nil {
		o = CreateStrObjectByStr(s, c.Argv[2])
		c.Db.Set(c.Argv[1], o)
	} else {
		if !CheckRType(o, OBJ_RTYPE_STR) {
			return
		}
		AppendStrObject(s, o, c.Argv[2])
		// addReply()
	}
}





