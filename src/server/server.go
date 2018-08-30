package server

import (
	. "redigo/src/structure"
	. "redigo/src/db"
	. "redigo/src/object"
	. "redigo/src/constant"
	. "redigo/src/networking"
	"sync"
	"fmt"
	"strconv"
	"bytes"
	"strings"
	"net"
	"time"
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

type Command struct {
	Name  string
	Arity int64
	Flags int64
	/* What keys should be loaded in background when calling this command? */
	FirstKey int64
	LastKey  int64
	KeyStep  int64
	Msec     int64
	Calls    int64
}

type Op struct {
	Argc   int64    // count of arguments
	Argv   []string // arguments of current command
	DbId   int64
	Target int64
	Cmd    *Command
}

type ClientBufferLimitsConfig struct {
	HardLimitBytes int64
	SoftLimitBytes int64
	SoftLimitTime time.Duration
}

type Server struct {
	Pid          int64
	PidFile      string
	ConfigFile   string
	ExecFile     string
	ExecArgv     []string
	Hz           int64 // serverCron() calls frequency in hertz
	Dbs          []*Db
	DbNum        int64
	Commands     map[interface{}]Command
	OrigCommands map[interface{}]Command

	UnixTime         time.Duration // UnixTime in millisecond
	LruClock         int64         // Clock for LRU eviction
	ShutdownNeedAsap bool

	CronLoops int64

	LoadModuleQueue *List // List of modules to load at startup.
	NextClientId    int64

	// Network
	Port           int64 // TCP listening port
	BindAddr       [CONFIG_BINDADDR_MAX]string
	BindAddrCount  int64  // Number of addresses in server.bindaddr[]
	UnixSocketPath string // UNIX socket path
	//IpFileDesc []string  // TCP socket file descriptors
	//SocketFileDesc string // Unix socket file descriptor
	CurrentClient       *Client
	Clients             *List // List of active clients
	ClientsMap          map[int64]*Client
	ClientsToClose      *List // Clients to close asynchronously
	ClientsPendingWrite *List // There is to write or install handler.
	ClientsUnblocked    *List //
	ClientObufLimits [CLIENT_TYPE_OBUF_COUNT]ClientBufferLimitsConfig
	MaxClients          int64
	ProtectedMode   bool // Don't accept external connections.
	Password        string
	RequirePassword bool
	TcpKeepAlive bool



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
	Dirty int64 // Changes to DB from the last save
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

	StatRejectedConn   int64
	StatConnCount      int64
	StatNetOutputBytes int64

	ConfigFlushAll bool
	mutex          sync.Mutex
}

func (s *Server) CreateClient(conn net.Conn) *Client {
	if conn != nil {
		if s.TcpKeepAlive {
			AnetSetTcpKeepALive(nil, conn.(*net.TCPConn), s.TcpKeepAlive)
		}
	}

	createTime := s.UnixTime
	var c = Client{
		Id:               0,
		Conn:             conn,
		Name:             "",
		QueryBuf:         make([]byte, PROTO_INLINE_MAX_SIZE),
		QueryBufPeak:     0,
		Argc:             0,                 // count of arguments
		Argv:             make([]string, 5), // arguments of current command
		Cmd:              nil,
		LastCmd:          nil,
		Reply:            ListCreate(),
		ReplySize:        0,
		SentSize:         0, // Amount of bytes already sent in the current buffer or object being sent.
		CreateTime:       createTime,
		LastInteraction:  createTime,
		Buf:              make([]byte, PROTO_REPLY_CHUNK_BYTES),
		BufPos:           0,
		SentLen:          0,
		Flags:            0,
		Node:             nil,
		PendingWriteNode: nil,
		UnblockedNode:    nil,
	}
	c.GetNextClientId(s)
	c.SelectDB(s, 0)
	return &c
}

func (s *Server) LinkClient(c *Client) {
	s.Clients.ListAddNodeTail(c)
	s.ClientsMap[c.Id] = c
	c.Node = s.Clients.ListTail()
	s.StatConnCount++
}

func (s *Server) UnLinkClient(c *Client) {
	if s.CurrentClient == c {
		s.CurrentClient = nil
	}
	if c.Conn != nil {
		s.Clients.ListDelNode(c.Node)
		c.Node = nil
		delete(s.ClientsMap, c.Id)
		s.StatConnCount--
		c.Conn = nil
	}
	if c.WithFlags(CLIENT_PENDING_WRITE) {
		s.ClientsPendingWrite.ListDelNode(c.PendingWriteNode)
		c.PendingWriteNode = nil
		c.DeleteFlags(CLIENT_PENDING_WRITE)
	}
	if c.WithFlags(CLIENT_UNBLOCKED) {
		s.ClientsUnblocked.ListDelNode(c.UnblockedNode)
		c.UnblockedNode = nil
		c.DeleteFlags(CLIENT_UNBLOCKED)
	}
}

func (s *Server) GetClientById(id int64) *Client {
	return s.ClientsMap[id]
}

func (s *Server) PrepareClientToWrite(c *Client) int64 {
	if c.WithFlags(CLIENT_LUA | CLIENT_MODULE) {
		return C_OK
	}

	if c.WithFlags(CLIENT_REPLY_OFF | CLIENT_REPLY_SKIP) {
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
	if c.Flags&CLIENT_CLOSE_ASAP != 0 || c.Flags&CLIENT_LUA != 0 {
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
	if len(str) != 0 || str[0] != '-' {
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

func (s *Server) AcceptUnixConn(conn net.Conn, flags int64) {
	c := s.CreateClient(conn)
	if c == nil {
		conn.Close()
	}
	if s.Clients.ListLength() > s.MaxClients {
		err := []byte("-ERR max number of clients reached\r\n")
		conn.Write(err)
		s.StatRejectedConn++
	}
	c.AddFlags(CLIENT_UNIX_SOCKET)
	s.LinkClient(c)
}

func (s *Server) AcceptTcpConn(conn net.Conn, flags int64, ip string) {

	c := s.CreateClient(conn)
	if c == nil {
		conn.Close()
	}
	if s.Clients.ListLength() > s.MaxClients {
		err := []byte("-ERR max number of clients reached\r\n")
		conn.Write(err)
		s.StatRejectedConn++
	}
	if s.ProtectedMode && s.BindAddrCount == 0 && !s.RequirePassword && ip != "" {
		err := []byte(
			`-DENIED Redis is running in protected mode because protected mode is enabled, no bind address was specified, no authentication password is requested to clients. In this mode 
connections are only accepted from the loopback interface. 

If you want to connect from external computers to Redis you may adopt one of the following solutions: 

1) Just disable protected mode sending the command 'CONFIG SET protected-mode no' from the loopback interface by connecting to Redis from the same host the server is running, however MAKE SURE Redis is not publicly accessible from internet if you do so. Use CONFIG REWRITE to make this change permanent.
2) Alternatively you can just disable the protected mode by editing the Redis configuration file, and setting the protectedmode option to 'no', and then restarting the server.
3) If you started the server manually just for testing, restart it with the '--protected-mode no' option.
4) Setup a bind address or an authentication password. 

NOTE: You only need to do one of the above things in order for the server to start accepting connections from the outside.\r\n`)
		conn.Write(err)
		s.StatRejectedConn++
	}
	c.AddFlags(0)
	s.LinkClient(c)
}

// Write data in output buffers to client.
func (s *Server) WriteToClient(c *Client) (written int64) {
	for c.HasPendingReplies() {
		if c.BufPos > 0 {
			n, err := c.Write(c.Buf)
			if err == nil {
				if n <= 0 {
					break
				}
				c.SentLen += int64(n)
				written += n
			}
			if c.SentLen == c.BufPos {
				c.SentLen = 0
				c.BufPos = 0
			}
		} else {
			str := c.Reply.ListHead().Value.(*string)
			length := int64(len(*str))
			if length == 0 {
				c.Reply.ListDelNode(c.Reply.ListHead())
			}
			n, err := c.Write([]byte(*str))
			if err == nil {
				if n <= 0 {
					break
				}
				c.SentLen += int64(n)
				written += n
			}
			if c.SentLen == length {
				c.Reply.ListDelNode(c.Reply.ListHead())
				c.SentLen = 0
				c.ReplySize -= length
				if c.Reply.ListLength() == 0 {
					c.ReplySize = 0
				}
			}
		}
		if written > NET_MAX_WRITES_PER_EVENT {
			break
		}
	}
	s.StatNetOutputBytes += written
	if written > 0 {
		if !c.WithFlags(CLIENT_MASTER) {
			c.LastInteraction = s.UnixTime
		}
	}
	return written
}

func (s *Server) ReadQueryFromClient(c *Client) {

}

func (s *Server) ProcessInlineBuffer(c *Client) int64 {
	// Search for end of line
	newline := bytes.IndexByte(c.QueryBuf,'\n')

	if newline == -1 {
		if len(c.QueryBuf) > PROTO_INLINE_MAX_SIZE {
			s.AddReplyError(c, "Protocol error: too big inline request")
		}
		return C_ERR
	}
	if newline != 0 && newline != len(c.QueryBuf) && c.QueryBuf[newline-1] == '\r' {
		// Handle the \r\n case.
		newline--
	}
	queryLen := newline
	aux := string(c.QueryBuf[0:newline])
	argv := strings.Split(aux, )

}

func (s *Server) ProcessInputBuffer(c *Client) {

}

func (s *Server) ProcessMultibulkBuffer(c *Client) {

}

func (s *Server) SetProtocolError(err *string, c *Client, pos int64) {

}

func (s *Server) GetAllClientInfoString(ctype int64) string {
	str := bytes.Buffer{}
	listIter := s.Clients.ListGetIterator(ITERATION_DIRECTION_INORDER)
	ln := listIter.ListNext()
	for ln != nil {
		c := ln.Value.(*Client)
		if ctype != -1 && c.GetClientType() != ctype {
			continue
		}
		str.WriteString(c.CatClientInfoString(s))
		str.WriteByte('\n')
	}
	return str.String()
}

func SplitArgs(query *[]byte, argc int64) string{
	
}