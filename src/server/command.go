package server

import (
	. "redigo/src/object"
	. "redigo/src/constant"
	"strings"
	"strconv"
)

/* SET key value [NX] [XX] [EX <seconds>] [PX <milliseconds>] */
// NX - not exist
// XX - exist
// EX - expire in seconds
// PX - expire in milliseconds
func SetGenericCommand(s *Server, c *Client, flags int64, key string) {
	if (flags&OBJ_SET_NX != 0 && c.Db.Exist(key)) ||
		(flags&OBJ_SET_XX != 0 && !c.Db.Exist(key)) {
		// addReply
		return
	}
	o := CreateStrObjectByStr(s, c.Argv[2])
	c.Db.Set(key, o)
	s.Dirty++

	//if expire

	// addReply
}

func SetCommand(s *Server, c *Client) {
	flags := OBJ_SET_NO_FLAGS
	for j := int64(3); j < c.Argc; j++ {
		a := strings.ToUpper(c.Argv[j])

		if a == "NX" && (flags&OBJ_SET_XX) != 0 {
			flags |= OBJ_SET_NX
		} else if a == "XX" && (flags&OBJ_SET_NX) != 0 {
			flags |= OBJ_SET_XX
		} else {
			// addReply
			return
		}
		// expire is not implemented now, so end here
	}

	SetGenericCommand(s, c, int64(flags), c.Argv[1])
}

func SetNxCommand(s *Server, c *Client) {
	SetGenericCommand(s, c, OBJ_SET_NX, c.Argv[1])
}

func SetXxCommand(s *Server, c *Client) {
	SetGenericCommand(s, c, OBJ_SET_XX, c.Argv[1])
}

func FlushAllCommand(s *Server, c *Client) {
	if s.ConfigFlushAll {
		c.Db.FlushAll()
		// addReply
	} else {
		// addReplyError(c, "cannot flush all")
	}
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
		// addReplyError(c, "increment or decrement would overflow")
		return
	}
	ReplaceStrObjectByInt(s, o, &oldValue, &value)
}

func IncrCommand(s *Server, c *Client) {
	IncrDecrCommand(s, c, 1)
}

func DecrCommand(s *Server, c *Client) {
	IncrDecrCommand(s, c, -1)
}

func IncrByCommand(s *Server, c *Client) {
	incr, err := strconv.ParseInt(c.Argv[2], 10, 64)
	if err != nil {
		return
	}
	IncrDecrCommand(s, c, incr)
}

func DecrByCommand(s *Server, c *Client) {
	decr, err := strconv.ParseInt(c.Argv[2], 10, 64)
	if err != nil {
		return
	}
	IncrDecrCommand(s, c, -decr)
}

func StrLenCommand(s *Server, c *Client) {
	o := s.DbGetOrReply(c, c.Argv[1], s.Shared.NullBulk)
	if o == nil {
		return
	}
	//str, err := GetStrObjectValueString(o)
	_, err := GetStrObjectValueString(o.(*StrObject))
	if err == nil {
		return
	}
	// addReply(len(str))
}

// Cat strings
func AppendCommand(s *Server, c *Client) {
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

func DbGetOrReply(s *Server, c *Client, key string, reply string) IObject {
	o := c.Db.Get(c.Argv[1])
	if o == nil {
		//addReply
	}
	return o
}

func GetGenericCommand(s *Server, c *Client) int64 {
	o := DbGetOrReply(s, c, c.Argv[1], s.Shared.NullBulk)
	if o == nil {
		return C_OK
	}
	if !CheckRType(o, OBJ_RTYPE_STR) {
		// addReply(c, s.Shared.WrongType)
		return C_ERR
	} else {
		// addReply
		return C_OK
	}
}

func GetCommand(s *Server, c *Client) {
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
		// addReplyError(c,"wrong number of arguments for MSET")
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
	// addReply(c, s.Shared.CommandOne)
}

func MSetCommand(s *Server, c *Client) {
	MSetGenericCommand(s, c, OBJ_SET_NO_FLAGS)
}

func MSetNxCommand(s *Server, c *Client) {
	MSetGenericCommand(s, c, OBJ_SET_NX)
}

func MGetCommand(s *Server, c *Client) {
	// addReplyMultBulk()...
	for j := 1; j < len(c.Argv); j++ {
		o := c.Db.Get(c.Argv[j])
		if o == nil {
			// addReply(c, s.Shared.NullBulk)
		} else {
			if !CheckRType(o, OBJ_RTYPE_STR) {
				// addReply(c, s.Shared.NullBulk)
			} else {
				// addReplyBulk(c, o)
			}
		}
	}
}

func AuthCommand(s *Server, c *Client) {
	if s.RequirePassword==nil {
		s.AddReplyError(c, "Client sent AUTH, but no password is set")
	} else if c.Argv[1] == *s.RequirePassword {
		c.Authenticated = 1
		s.AddReply(c, s.Shared.Ok)
	} else {
		c.Authenticated = 0
		s.AddReplyError(c, "invalid password")
	}
}
