package server

import (
	"strings"
	"strconv"
	"time"
	"sync/atomic"
)

type CommandProcess func(c *KiwiClient)
type CommandGetKeysProcess func(cmd *Command, argv []string, argc int, numkeys []int)

type Command struct {
	Name          string
	Process       CommandProcess
	Arity         int // -N means >= N, N means = N
	CharFlags     string
	Flags         int
	GetKeyProcess CommandGetKeysProcess
	/* What keys should be loaded in background when calling this command? */
	FirstKey bool
	LastKey  bool
	KeyStep  int
	Duration time.Duration
	Calls    int
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
	//{"randomkey", RandomKeyCommand, 1, "rR", 0, nil, false, false, 0, 0, 0},
	{"select", SelectCommand, 2, "lF", 0, nil, false, false, 0, 0, 0},
	{"flushall", FlushAllCommand, -1, "w", 0, nil, false, false, 0, 0, 0},
	{"flushall", FlushAllCommand, -1, "w", 0, nil, false, false, 0, 0, 0},

}

func PopulateCommandTable() {
	for k, cmd := range CommandTable {
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
		kiwiS.Commands[cmd.Name] = &CommandTable[k]
		kiwiS.OrigCommands[cmd.Name] = &CommandTable[k]
	}
}

func (cmd *Command) WithFlags(flags int) bool {
	return cmd.Flags&flags != 0
}

func (cmd *Command) AddFlags(flags int) {
	cmd.Flags |= flags
}

func (cmd *Command) DeleteFlags(flags int) {
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
func SetGenericCommand(c *KiwiClient, flags int, key string, okReply string, abortReply string) {
	// fmt.Println("SetGenericCommand")
	if (flags&OBJ_SET_NX != 0 && c.Db.Exist(key)) || (flags&OBJ_SET_XX != 0 && !c.Db.Exist(key)) {
		if abortReply != "" {
			AddReply(c, kiwiS.Shared.Zero)
		} else {
			AddReply(c, abortReply)
		}
		return
	}
	o := CreateStrObjectByStr(c.Argv[2])
	c.Db.Set(key, o)
	atomic.AddInt64(&kiwiS.Dirty, 1)
	if okReply == "" {
		AddReply(c, kiwiS.Shared.One)
	} else {
		AddReply(c, okReply)
	}
}

var SetCommand CommandProcess = func(c *KiwiClient) {
	// fmt.Println("SetCommand")
	flags := OBJ_SET_NO_FLAGS
	for j := 3; j < c.Argc; j++ {
		a := strings.ToUpper(c.Argv[j])

		if a == "NX" && (flags&OBJ_SET_XX) != 0 {
			flags |= OBJ_SET_NX
		} else if a == "XX" && (flags&OBJ_SET_NX) != 0 {
			flags |= OBJ_SET_XX
		} else if a == "EX" && flags&OBJ_SET_PX != 0 {
			flags |= OBJ_SET_EX
		} else if a == "PX" && flags&OBJ_SET_EX != 0 {
			flags |= OBJ_SET_PX
		} else {
			AddReply(c, kiwiS.Shared.SyntaxErr)
			return
		}
		// TODO expire is not implemented now, so end here
	}
	SetGenericCommand(c, flags, c.Argv[1], kiwiS.Shared.One, kiwiS.Shared.Zero)
}

var SetNxCommand CommandProcess = func(c *KiwiClient) {
	SetGenericCommand(c, OBJ_SET_NX, c.Argv[1], kiwiS.Shared.One, kiwiS.Shared.Zero)
}

var SetExCommand CommandProcess = func(c *KiwiClient) {
	SetGenericCommand(c, OBJ_SET_XX, c.Argv[1], kiwiS.Shared.One, kiwiS.Shared.Zero)
}

var FlushAllCommand CommandProcess = func(c *KiwiClient) {
	if kiwiS.ConfigFlushAll {
		c.Db.FlushAll()
		AddReply(c, kiwiS.Shared.Ok)
		//	TODO update aof or rdb
	}
	atomic.AddInt64(&kiwiS.Dirty, 1)
}

var ExistsCommand CommandProcess = func(c *KiwiClient) {
	count := 0
	for j := 1; j < c.Argc; j++ {
		if c.Db.Exist(c.Argv[1]) {
			count++
		}
	}
	AddReplyInt(c, count)
	// expire
	// addReply()
}

func IncrDecrCommand(c *KiwiClient, incr int) {
	o := c.Db.Get(c.Argv[1]).(*StrObject)
	if o == nil {
		o = CreateStrObjectByInt(incr)
		c.Db.Set(c.Argv[1], o)
	}
	if !IsStrObjectInt(o) {
		return
	}
	value := *o.Value.(*int)
	oldValue := value
	value += incr
	if IsOverflowInt(oldValue, incr) {
		AddReplyError(c, "increment or decrement would overflow")
		return
	}
	ReplaceStrObjectByInt(o, &oldValue, &value)
	atomic.AddInt64(&kiwiS.Dirty, 1)
	AddReplyInt(c, value)
}

var IncrCommand CommandProcess = func(c *KiwiClient) {
	IncrDecrCommand(c, 1)
}

var DecrCommand CommandProcess = func(c *KiwiClient) {
	IncrDecrCommand(c, -1)
}

var IncrByCommand CommandProcess = func(c *KiwiClient) {
	incr, err := strconv.Atoi(c.Argv[2])
	if err != nil {
		return
	}
	IncrDecrCommand(c, incr)
}

var DecrByCommand CommandProcess = func(c *KiwiClient) {
	decr, err := strconv.Atoi(c.Argv[2])
	if err != nil {
		return
	}
	IncrDecrCommand(c, -decr)
}

var StrLenCommand CommandProcess = func(c *KiwiClient) {
	o := DbGetOrReply(c, c.Argv[1], kiwiS.Shared.NullBulk)
	if o == nil {
		return
	}
	str, err := GetStrObjectValueString(o.(*StrObject))
	if err == nil {
		return
	}
	AddReplyInt(c, len(str))
}

// Cat strings
var AppendCommand CommandProcess = func(c *KiwiClient) {
	var length int
	o := c.Db.Get(c.Argv[1]).(*StrObject)
	if o == nil {
		o = CreateStrObjectByStr(c.Argv[2])
		c.Db.Set(c.Argv[1], o)
	} else {
		if !CheckOType(o, OBJ_RTYPE_STR) {
			return
		}
		_, length = AppendStrObject(o, c.Argv[2])
	}
	atomic.AddInt64(&kiwiS.Dirty, 1)
	AddReplyInt(c, length)
}

func DbGetOrReply(c *KiwiClient, key string, reply string) Objector {
	o := c.Db.Get(key)
	if o == nil {
		AddReply(c, reply)
	}
	return o
}

func GetGenericCommand(c *KiwiClient) int {
	o := DbGetOrReply(c, c.Argv[1], kiwiS.Shared.NullBulk)
	if o == nil {
		return C_OK
	}
	if !CheckOType(o, OBJ_RTYPE_STR) {
		AddReply(c, kiwiS.Shared.WrongTypeErr)
		return C_ERR
	} else {
		AddReplyBulkStrObj(c, o.(*StrObject))
		return C_OK
	}
}

var GetCommand CommandProcess = func(c *KiwiClient) {
	GetGenericCommand(c)
}

func GetSetCommand(c *KiwiClient) {
	if GetGenericCommand(c) == C_ERR {
		return
	}
	o := CreateStrObjectByStr(c.Argv[2])
	c.Db.Set(c.Argv[1], o)
	atomic.AddInt64(&kiwiS.Dirty, 1)
}

func MSetGenericCommand(c *KiwiClient, flags int) {
	if c.Argc%2 == 0 {
		AddReplyError(c, "wrong number of arguments for MSET")
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
			AddReply(c, kiwiS.Shared.Zero)
			return
		}
	}
	for j := 1; j < len(c.Argv); j += 2 {
		o := CreateStrObjectByStr(c.Argv[j+1])
		c.Db.Set(c.Argv[j], o)
	}
	atomic.AddInt64(&kiwiS.Dirty, int64((c.Argc-1)/2))
	if flags&OBJ_SET_NX != 0 {
		AddReply(c, kiwiS.Shared.One)
	} else {
		AddReply(c, kiwiS.Shared.Ok)
	}
}

var MSetCommand CommandProcess = func(c *KiwiClient) {
	MSetGenericCommand(c, OBJ_SET_NO_FLAGS)
}

var MSetNxCommand CommandProcess = func(c *KiwiClient) {
	MSetGenericCommand(c, OBJ_SET_NX)
}

var MGetCommand CommandProcess = func(c *KiwiClient) {
	AddReplyMultiBulkLen(c, c.Argc-1)
	for j := 1; j < len(c.Argv); j++ {
		o := c.Db.Get(c.Argv[j]).(*StrObject)
		if o == nil {
			AddReply(c, kiwiS.Shared.NullBulk)
		} else {
			if !CheckOType(o, OBJ_RTYPE_STR) {
				AddReply(c, kiwiS.Shared.NullBulk)
			} else {
				AddReplyBulkStrObj(c, o)
			}
		}
	}
}

var AuthCommand CommandProcess = func(c *KiwiClient) {
	if kiwiS.RequirePassword == nil {
		AddReplyError(c, "KiwiClient sent AUTH, but no password is set")
	} else if c.Argv[1] == *kiwiS.RequirePassword {
		c.Authenticated = 1
		AddReply(c, kiwiS.Shared.Ok)
	} else {
		c.Authenticated = 0
		AddReplyError(c, "invalid password")
	}
}

var DeleteCommand CommandProcess = func(c *KiwiClient) {
	DeleteGenericCommand(c, false)
}

func DeleteGenericCommand(c *KiwiClient, lazy bool) {
	count := 0
	for j := 1; j < c.Argc; j++ {
		var deleted bool
		if lazy {
			deleted = DbDeleteAsync(c, c.Argv[j])
		} else {
			deleted = DbDeleteSync(c, c.Argv[j])
		}
		if deleted {
			count++
			atomic.AddInt64(&kiwiS.Dirty, 1)
		}
	}
	AddReplyInt(c, count)
}

var SelectCommand CommandProcess = func(c *KiwiClient) {
	i, err := strconv.Atoi(c.Argv[1])
	if err != nil {
		AddReplyError(c, "invalid DB index")
	} else {
		if SelectDB(c, i) == C_ERR {
			AddReplyError(c, "DB index is out of range")
		} else {
			AddReply(c, kiwiS.Shared.Ok)
		}
	}
}
//
//var RandomKeyCommand CommandProcess = func(c *KiwiClient) {
//	key, value := c.Db.RandGet()
//	if key == "" && value == nil {
//		AddReply( c, kiwiS.Shared.NullBulk)
//	} else {
//		AddReplyBulkStr( c, key)
//	}
//}

//var ExpireCommand CommandProcess = func(c *KiwiClient) {
//	key, timeStr := c.Argv[0], c.Argv[1]
//	timeMs := strconv.
//
//}