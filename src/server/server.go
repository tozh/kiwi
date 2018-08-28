package server

import (
	."redigo/src/structure"
	."redigo/src/db"
	."redigo/src/object"
	."redigo/src/constant"
	"sync"
	"fmt"
	"strconv"
	"bytes"
	"strings"
	"net"
)

//type Object struct {
//	ObjType int64
//	Encoding int64
//	Lru int64
//	RefCount int64
//	Ptr interface{}
//}
//
//type RedisDb struct {
//	Dict map[string]interface{}
//	Expires map[*Object]int64
//	Id int64
//	AvgTTL int64
//	DefragLater *List
//}

type RedisCommand struct {
	Name string
	Arity int64
	Flags int64
	/* What keys should be loaded in background when calling this command? */
	FirstKey int64
	LastKey int64
	KeyStep int64
	Msec int64
	Calls int64
}

type Op struct {
	Argc int64       // count of arguments
	Argv []string // arguments of current command
	DbId int64
	Target int64
	Cmd *RedisCommand
}

type Server struct {
	Pid int64
	PidFile string
	ConfigFile string
	ExecFile string
	ExecArgv []string
	Hz int64		// serverCron() calls frequency in hertz
	Dbs []*Db
	DbNum int64
	Commands map[interface{}]RedisCommand
	OrigCommands map[interface{}]RedisCommand

	UnixTimeInMs int64 // UnixTime in millisecond
	LruClock int64 // Clock for LRU eviction
	ShutdownNeedAsap bool

	CronLoops int64

	LoadModuleQueue *List  // List of modules to load at startup.
	NextClientId int64

	// Network
	Port int64  // TCP listening port
	BindAddr []string
	UnixSocket string  // UNIX socket path
	IpFileDesc []string  // TCP socket file descriptors
	SocketFileDesc string // Unix socket file descriptor
	Clients *List  // List of active clients
	ClientsToClose *List  // Clients to close asynchronously
	ClientsPendingWrite *List  // There is to write or install handler.
	//clientsToClose *List  // Clients to close asynchronously

	//Loading bool  // Server is loading date from disk if true
	//LoadingTotalSize int64
	//LoadingLoadedSize int64
	//LoadingStartTime int64
	//
	//
	//// Configuration
	//MaxIdleTimeSec int64  // Client timeout in seconds
	//TcpKeepAlive bool
	//
	//
	Dirty int64  // Changes to DB from the last save
	// DirtyBeforeBgSave  //  Used to restore dirty on failed BGSAVE
	//// AOF persistence
	//AofState int64  //  AOF(ON|OFF|WAIT_REWRITE)
	//AofChildPid int64  // PID if rewriting process
	//AofFileSync int64  // Kind of fsync() policy
	//AofFileName string
	//AofRewirtePercent int64  // Rewrite AOF if % growth is > M and...
	//AofRewriteMinSize int64
	//AofRewriteMaxSize int64
	//AofRewriteScheduled bool  // Rewrite once BGSAVE terminates.
	//AofRewriteBufBlocks *List  // Hold changes during AOF rewrites
	//AofBuf string  // Aof buffer
	//AofFileDesc int64  // File descriptor of currently selected AOF file
	//AofSelectedDb bool  // Currently selected DB in AOF
	//AofPropagate []*RedisOp  // Additional command to propagate.


	//// RDB persistence
	//
	//// Logging
	//LogFile string
	//SysLogEnable bool
	//
	//
	//// Zip structure config
	//HashMaxZiplistEntries int64
	//HashMaxZiplistValue int64
	//SetMaxInsetEntrie int64
	//ZSetMaxZiplistEntries int64
	//ZSetMaxZiplistvalue int64

	Shared SharedObjects

	ConfigFlushAll bool
	mutex sync.Mutex
}

func (s *Server) CreateClient(conn *net.Conn) *Client {
	createTime := s.UnixTimeInMs
	var c = Client{
		Id: 0,
		Conn: conn,
		Name: "",
		QueryBuf: "",
		QueryBufPeak: 0,
		Argc: 0,       // count of arguments
		Argv: make([]string, 5), // arguments of current command
		Cmd: nil,
		LastCmd: nil,
		Reply: ListCreate(),
		ReplySize: 0,
		SentSize: 0, // Amount of bytes already sent in the current buffer or object being sent.
		CreateTime: createTime,
		LastInteraction: createTime,
		Buf: make([]byte, PROTO_REPLY_CHUNK_BYTES),
		BufPos:0,
		SentLen:0,
		Flags: 0,
	}
	c.GetNextClientId(s)
	c.SelectDB(s, 0)
	return &c
}

func (s *Server) PrepareClientToWrite(c *Client) int64 {
	if c.WithFlags(CLIENT_LUA|CLIENT_MODULE) {
		return C_OK
	}

	if c.WithFlags(CLIENT_REPLY_OFF|CLIENT_REPLY_SKIP) {
		return C_ERR
	}

	if c.WithFlags(CLIENT_MASTER) && !c.WithFlags(CLIENT_MASTER_FORCE_REPLY) {
		return C_ERR
	}
	if c.Conn == nil {
		// Fake client for AOF loading.
		return C_ERR
	}

	if !c.HasPendingReplies() && !(c.WithFlags(CLIENT_PENDING_WRITE)) {
		c.AddFlags(CLIENT_PENDING_WRITE)
		s.ClientsPendingWrite.ListAddNodeTail(c)
	}
	return C_OK
}


func (s *Server) CloseClientAsync(c *Client) {
	if c.Flags & CLIENT_CLOSE_ASAP != 0 || c.Flags & CLIENT_LUA != 0 {
		return
	}
	c.AddFlags(CLIENT_CLOSE_ASAP)
	s.ClientsToClose.ListAddNodeTail(c)
}


func (s *Server) AddReply(c *Client, str string) {
	if s.PrepareClientToWrite(c) != C_OK {
		return
	}
	if c.AddReplyToBuffer(str) != C_OK {
		c.AddReplyStringToList(str)
	}
}

func (s *Server) AddReplyStrObj(c *Client, o *StrObject) {
	if !CheckRType(o, OBJ_RTYPE_STR) {
		return
	}
	str, err := GetStrObjectValueString(o)
	if err == nil {
		s.AddReply(c, str)
	} else {
		return
	}
}

func (s *Server) AddReplyError(c *Client, str string) {
	if len(str) !=0 || str[0] != '-' {
		s.AddReply(c, "-ERR ")
	}
	s.AddReply(c, str)
	s.AddReply(c, "\r\n")
}

func (s *Server) AddReplyErrorFormat(c *Client, format string, a ...interface{}) {
	str := fmt.Sprintf(format, a)
	s.AddReplyError(c, str)
}

func (s *Server) AddReplyStatus(c *Client, str string) {
	s.AddReply(c, "+")
	s.AddReply(c, str)
	s.AddReply(c, "\r\n")
}

func (s *Server) AddReplyStatusFormat(c *Client, format string, a ...interface{}) {
	str := fmt.Sprintf(format, a)
	s.AddReplyStatus(c, str)
}

//func (s *Server) AddReplyHelp(c *Client, help []string) {
//	cmd := c.Argv[0]
//	s.AddReplyStatusFormat(c, "%s <subcommand> arg arg ... arg. Subcommands are:", cmd)
//	for _, h := range help {
//		s.AddReplyStatus(c, h)
//	}
//}

func (s *Server) AddReplyIntWithPrifix(c *Client, i int64, prefix byte) {
	/* Things like $3\r\n or *2\r\n are emitted very often by the protocol
	so we have a few shared objects to use if the integer is small
	like it is most of the times. */
	if prefix == '*' && i >= 0 && i < SHARED_BULKHDR_LEN {
		s.AddReply(c, s.Shared.MultiBulkHDR[i])
	} else if prefix == '$' && i >= 0 && i < SHARED_BULKHDR_LEN {
		s.AddReply(c, s.Shared.MultiBulkHDR[i])
	} else {
		str := strconv.FormatInt(i, 10)
		buf := bytes.Buffer{}
		buf.WriteByte(prefix)
		buf.WriteString(str)
		buf.WriteByte('\r')
		buf.WriteByte('\n')
		s.AddReply(c, buf.String())
	}
}

func (s *Server) AddReplyInt(c *Client, i int64) {
	if i == 0 {
		s.AddReply(c, s.Shared.Zero)
	} else if i == 1 {
		s.AddReply(c, s.Shared.One)
	} else {
		s.AddReplyIntWithPrifix(c, i, ':')
	}
}

func (s *Server) AddReplyMultiBulkLength(c *Client, length int64) {
	s.AddReplyIntWithPrifix(c, length, '*')
}

/* Create the length prefix of a bulk reply, example: $2234 */
func (s *Server) AddReplyBulkLengthString(c *Client, str string) {
	length := int64(len(str))
	s.AddReplyIntWithPrifix(c, length, '$')
}

func (s *Server) AddReplyBulkLengthStrObj(c *Client, o *StrObject) {
	if !CheckRType(o, OBJ_RTYPE_STR) {
		return
	}
	str, err := GetStrObjectValueString(o)
	if err == nil {
		s.AddReplyBulkLengthString(c, str)
	} else {
		return
	}
}

func (s *Server) AddReplyBulk(c *Client, o *StrObject) {
	s.AddReplyBulkLengthStrObj(c, o)
	s.AddReplyStrObj(c, o)
	s.AddReply(c, s.Shared.Crlf)
}

func (s *Server) AddReplyBulkString(c *Client, str string) {
	if str == "" {
		s.AddReply(c, s.Shared.NullBulk)
	} else {
		s.AddReplyBulkLengthString(c, str)
		s.AddReply(c, str)
		s.AddReply(c, s.Shared.Crlf)
	}
}

func (s *Server) AddReplyBulkInt(c *Client, i int64) {
	str := strconv.FormatInt(i, 10)
	s.AddReplyBulkString(c, str)
}

func (s *Server) AddReplySubcommandSyntaxError(c *Client) {
	cmd := c.Argv[0]
	s.AddReplyErrorFormat(c, "Unknown subcommand or wrong number of arguments for '%s'. Try %s HELP.", cmd, strings.ToUpper(cmd))
}

func (s *Server) AcceptCommonHandler(conn *net.Conn, flags int64, ip string) {

}








