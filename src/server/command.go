package server

import (
	. "redigo/src/constant"
	"strings"
	"strconv"
	"time"
)

type CommandProcess func(s *Server, c *Client)
type CommandGetKeysProcess func(cmd *Command, argv []string, argc int64, numkeys []int64)

type Command struct {
	Name          string
	Process       CommandProcess
	Arity         int64 // -N means >= N, N means = N
	CharFlags     string
	Flags         int64
	GetKeyProcess CommandGetKeysProcess
	/* What keys should be loaded in background when calling this command? */
	FirstKey bool
	LastKey  bool
	KeyStep  int64
	Duration time.Duration
	Calls    int64
}

var CommandTable = []Command{
	{"get", GetCommand, 2, "rF", 0, nil, true, true, 1, 0, 0},
	{"set", SetCommand, -3, "wm", 0, nil, true, true, 1, 0, 0},
	{"setnx", SetNxCommand, 3, "wmF", 0, nil, true, true, 1, 0, 0},
	{"setex", SetExCommand, 4, "wm", 0, nil, true, true, 1, 0, 0},
	{"append", AppendCommand, 3, "wm", 0, nil, true, true, 1, 0, 0},
	{"strlen", StrLenCommand, 2, "rF", 0, nil, true, true, 1, 0, 0},
	{"del", DeleteCommand, -2, "w", 0, nil, true, false, 1, 0, 0},
	//{"unlink", UnlinkCommand, -2, "wF", 0, nil, true, false, 1, 0 , 0},
	{"exists", ExistsCommand, -2, "rF", 0, nil, true, false, 1, 0, 0},
	{"incr", IncrCommand, 2, "wmF", 0, nil, true, true, 1, 0, 0},
	{"decr", DecrCommand, 2, "wmF", 0, nil, true, true, 1, 0, 0},
	{"mget", MGetCommand, -2, "rF", 0, nil, true, false, 1, 0, 0},
	{"mset", MSetCommand, -3, "wm", 0, nil, true, true, 2, 0, 0},
	{"msetnx", GetCommand, 2, "wm", 0, nil, true, true, 2, 0, 0},
	{"randomkey", RandomKeyCommand, 1, "rR", 0, nil, false, false, 0, 0, 0},
	{"select", SelectCommand, 2, "lF", 0, nil, false, false, 0, 0, 0},
	{"flushall", FlushAllCommand, -1, "w", 0, nil, false, false, 0, 0, 0},
}

func PopulateCommandTable(s *Server) {
	for _, cmd := range CommandTable {
		for i := 0; i < len(cmd.CharFlags); i++ {
			switch cmd.CharFlags[i] {
			case 'w':
				cmd.Flags |= CMD_WRITE
			case 'r':
				cmd.Flags |= CMD_READONLY
			case 'm':
				cmd.Flags |= CMD_DENYOOM
			case 'a':
				cmd.Flags |= CMD_ADMIN
			case 'p':
				cmd.Flags |= CMD_PUBSUB
			case 's':
				cmd.Flags |= CMD_NOSCRIPT
			case 'R':
				cmd.Flags |= CMD_RANDOM
			case 'S':
				cmd.Flags |= CMD_SORT_FOR_SCRIPT
			case 'l':
				cmd.Flags |= CMD_LOADING
			case 't':
				cmd.Flags |= CMD_STALE
			case 'M':
				cmd.Flags |= CMD_SKIP_MONITOR
			case 'k':
				cmd.Flags |= CMD_ASKING
			case 'F':
				cmd.Flags |= CMD_FAST
			default:
				panic("Unsupported command flag")
			}
		}
		s.Commands[cmd.Name] = &cmd
		s.OrigCommands[cmd.Name] = &cmd
	}

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
			AddReply(s, c, s.Shared.NullBulk)
		} else {
			AddReply(s, c, abort_reply)
		}
		return
	}
	o := CreateStrObjectByStr(s, c.Argv[2])
	c.Db.Set(key, o)
	s.Dirty++
	if ok_reply != "" {
		AddReply(s, c, s.Shared.Ok)
	} else {
		AddReply(s, c, ok_reply)
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
			AddReply(s, c, s.Shared.SyntaxErr)
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
		AddReply(s, c, s.Shared.Ok)
		//	TODO update aof or rdb
	}
	s.Dirty++
}

var ExistsCommand CommandProcess = func(s *Server, c *Client) {
	count := int64(0)
	for j := int64(1); j < c.Argc; j++ {
		if c.Db.Exist(c.Argv[1]) {
			count++
		}
	}
	AddReplyInt(s, c, count)
	// expire
	// addReply()
}

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
		AddReplyError(s, c, "increment or decrement would overflow")
		return
	}
	ReplaceStrObjectByInt(s, o, &oldValue, &value)
	s.Dirty++
	AddReplyInt(s, c, value)
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
	AddReplyInt(s, c, int64(len(str)))
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
	AddReplyInt(s, c, length)
}

func DbGetOrReply(s *Server, c *Client, key string, reply string) Objector {
	o := c.Db.Get(key)
	if o == nil {
		AddReply(s, c, reply)
	}
	return o
}

func GetGenericCommand(s *Server, c *Client) int64 {
	o := DbGetOrReply(s, c, c.Argv[1], s.Shared.NullBulk)
	if o == nil {
		return C_OK
	}
	if !CheckRType(o, OBJ_RTYPE_STR) {
		AddReply(s, c, s.Shared.WrongTypeErr)
		return C_ERR
	} else {
		AddReplyBulk(s, c, o.(*StrObject))
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
		AddReplyError(s, c, "wrong number of arguments for MSET")
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
			AddReply(s, c, s.Shared.Zero)
			return
		}
	}
	for j := 1; j < len(c.Argv); j += 2 {
		o := CreateStrObjectByStr(s, c.Argv[j+1])
		c.Db.Set(c.Argv[j], o)
	}
	s.Dirty += (c.Argc - 1) / 2
	if flags&OBJ_SET_NX != 0 {
		AddReply(s, c, s.Shared.One)
	} else {
		AddReply(s, c, s.Shared.Ok)
	}
}

var MSetCommand CommandProcess = func(s *Server, c *Client) {
	MSetGenericCommand(s, c, OBJ_SET_NO_FLAGS)
}

var MSetNxCommand CommandProcess = func(s *Server, c *Client) {
	MSetGenericCommand(s, c, OBJ_SET_NX)
}

var MGetCommand CommandProcess = func(s *Server, c *Client) {
	AddReplyMultiBulkLength(s, c, c.Argc-1)
	for j := 1; j < len(c.Argv); j++ {
		o := c.Db.Get(c.Argv[j]).(*StrObject)
		if o == nil {
			AddReply(s, c, s.Shared.NullBulk)
		} else {
			if !CheckRType(o, OBJ_RTYPE_STR) {
				AddReply(s, c, s.Shared.NullBulk)
			} else {
				AddReplyBulk(s, c, o)
			}
		}
	}
}

var AuthCommand CommandProcess = func(s *Server, c *Client) {
	if s.RequirePassword == nil {
		AddReplyError(s, c, "Client sent AUTH, but no password is set")
	} else if c.Argv[1] == *s.RequirePassword {
		c.Authenticated = 1
		AddReply(s, c, s.Shared.Ok)
	} else {
		c.Authenticated = 0
		AddReplyError(s, c, "invalid password")
	}
}

var DeleteCommand CommandProcess = func(s *Server, c *Client) {
	DeleteGenericCommand(s, c, false)
}

func DeleteGenericCommand(s *Server, c *Client, lazy bool) {
	count := int64(0)
	for j := int64(1); j < c.Argc; j++ {
		var deleted bool
		if lazy {
			deleted = DbDeleteAsync(s, c, c.Argv[j])
		} else {
			deleted = DbDeleteSync(s, c, c.Argv[j])
		}
		if deleted {
			count++
			s.Dirty++
		}
	}
	AddReplyInt(s, c, count)
}

var SelectCommand CommandProcess = func(s *Server, c *Client) {
	i, err := strconv.ParseInt(c.Argv[1], 10, 64)
	if err != nil {
		AddReplyError(s, c, "invalid DB index")
	} else {
		if SelectDB(s, c, i) == C_ERR {
			AddReplyError(s, c, "DB index is out of range")
		} else {
			AddReply(s, c, s.Shared.Ok)
		}
	}
}

var RandomKeyCommand CommandProcess = func(s *Server, c *Client) {
	key, value := c.Db.RandGet()
	if key == "" && value == nil {
		AddReply(s, c, s.Shared.NullBulk)
	} else {
		AddReplyBulkString(s, c, key)
	}
}
