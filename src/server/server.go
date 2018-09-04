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

type CommandProcess func(s *Server, c *Client)

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
	Process  CommandProcess
	//Process  func(s *Server, c *Client)
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
	SoftLimitTime  time.Duration
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
	Commands     map[string]*Command
	OrigCommands map[string]*Command

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
	ClientObufLimits    [CLIENT_TYPE_OBUF_COUNT]ClientBufferLimitsConfig
	MaxClients          int64
	ProtectedMode       bool // Don't accept external connections.
	Password            string
	RequirePassword     *string
	TcpKeepAlive        bool
	Verbosity           int64 // loglevel in redis.conf
	ProtoMaxBulkLen     int64
	ClientsPaused       bool
	ClientsPauseEndTime time.Duration
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
	if s.ProtectedMode && s.BindAddrCount == 0 && s.RequirePassword == nil && ip != "" {
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

/* Like processMultibulkBuffer(), but for the inline protocol instead of RESP,
 * this function consumes the client query buffer and creates a command ready
 * to be executed inside the client structure. Returns C_OK if the command
 * is ready to be executed, or C_ERR if there is still protocol to read to
 * have a well formed command. The function also returns C_ERR when there is
 * a protocol error: in such a case the client structure is setup to reply
 * with the error and close the connection. */
func (s *Server) ProcessInlineBuffer(c *Client) int64 {
	// Search for end of line
	newline := bytes.IndexByte(c.QueryBuf, '\n')

	if newline == -1 {
		if len(c.QueryBuf) > PROTO_INLINE_MAX_SIZE {
			s.AddReplyError(c, "Protocol error: too big inline request")
			s.SetProtocolError("too big inline request", c, 0)
		}
		return C_ERR
	}
	if newline != 0 && newline != len(c.QueryBuf) && c.QueryBuf[newline-1] == '\r' {
		// Handle the \r\n case.
		newline--
	}
	/* Split the input buffer up to the \r\n */
	argvs := SplitArgs(c.QueryBuf[0:newline])
	if argvs == nil {
		s.AddReplyError(c, "Protocol error: unbalanced quotes in request")
		s.SetProtocolError("unbalanced quotes in inline request", c, 0)
		return C_ERR
	}

	// Leave data after the first line of the query in the buffer
	c.QueryBuf = c.QueryBuf[newline+2:]
	if len(argvs) != 0 {
		c.Argc = 0
		c.Argv = make([]string, len(argvs))
	}
	for index, argv := range argvs {
		if argv != "" {
			c.Argv[index] = argv
			c.Argc++
		}
	}
	return C_OK
}

/* Process the query buffer for client 'c', setting up the client argument
 * vector for command execution. Returns C_OK if after running the function
 * the client has a well-formed ready to be processed command, otherwise
 * C_ERR if there is still to read more buffer to get the full command.
 * The function also returns C_ERR when there is a protocol error: in such a
 * case the client structure is setup to reply with the error and close
 * the connection.
 *
 * This function is called if processInputBuffer() detects that the next
 * command is in RESP format, so the first byte in the command is found
 * to be '*'. Otherwise for inline commands processInlineBuffer() is called. */
func (s *Server) ProcessMultiBulkBuffer(c *Client) int64 {
	pos := 0
	if c.MultiBulkLen == 0 {
		newline := bytes.IndexByte(c.QueryBuf, '\r')
		if newline < 0 {
			if len(c.QueryBuf) > PROTO_INLINE_MAX_SIZE {
				s.AddReplyError(c, "Protocol error: too big multibulk count request")
				s.SetProtocolError("too big multibulk request", c, 0)
			}
			return C_ERR
		}
		if len(c.QueryBuf)-newline < 2 {
			// end with \r\n, so \r cannot be the last char
			return C_ERR
		}
		if c.QueryBuf[0] != '*' {
			return C_ERR
		}
		nulkNum, err := strconv.Atoi(string(c.QueryBuf[pos+1 : newline]))
		if err != nil || nulkNum > 1024*1024 {
			s.AddReplyError(c, "Protocol error: invalid multibulk length")
			s.SetProtocolError("invalid multibulk length", c, 0)
			return C_ERR
		}
		// pos start of bulks
		pos = newline + 2
		if nulkNum <= 0 {
			c.QueryBuf = c.QueryBuf[pos:]
			return C_OK
		}
		c.MultiBulkLen = int64(nulkNum)
		c.Argv = make([]string, c.MultiBulkLen)
	}
	if c.MultiBulkLen < 0 {
		return C_ERR
	}
	for c.MultiBulkLen > 0 {
		if c.BulkLen == -1 {
			// Read bulk length if unknown
			newline := bytes.IndexByte(c.QueryBuf, '\r')
			if newline < 0 {
				if len(c.QueryBuf) > PROTO_INLINE_MAX_SIZE {
					s.AddReplyError(c, "Protocol error: too big bulk count string")
					s.SetProtocolError("too big bulk count string", c, 0)
					return C_ERR
				}
				break
			}
			if len(c.QueryBuf)-newline < 2 {
				// end with \r\n, so \r cannot be the last char
				break
			}
			if c.QueryBuf[pos] != '$' {
				s.AddReplyError(c, fmt.Sprintf("Protocol error: expected '$', got '%c'", c.QueryBuf[pos]))
				s.SetProtocolError("expected $ but got something else", c, 0)
				return C_ERR
			}
			nulkNum, err := strconv.Atoi(string(c.QueryBuf[pos+1 : newline]))
			if err != nil || int64(nulkNum) > s.ProtoMaxBulkLen {
				s.AddReplyError(c, "Protocol error: invalid bulk length")
				s.SetProtocolError("invalid bulk length", c, 0)
				return C_ERR
			}
			pos = newline + 2
			if nulkNum >= PROTO_MBULK_BIG_ARG {
				/* If we are going to read a large object from network
				 * try to make it likely that it will start at c->querybuf
				 * boundary so that we can optimize object creation
				 * avoiding a large copy of data. */
				c.QueryBuf = c.QueryBuf[pos:]
				qblen := len(c.QueryBuf)
				pos = 0
				if qblen < nulkNum+2 {
					//	the only bulk
					c.QueryBuf = append(c.QueryBuf, make([]byte, nulkNum+2-qblen)...)
				}
				c.BulkLen = int64(nulkNum)
			}
			if int64(len(c.QueryBuf)-pos) < c.BulkLen+2 {
				break
			} else {
				if pos == 0 && c.BulkLen >= PROTO_MBULK_BIG_ARG && int64(len(c.QueryBuf)) == c.BulkLen+2 {
					c.Argv = append(c.Argv, string(c.QueryBuf[pos:c.BulkLen]))
					c.Argc++
				} else {
					c.Argv = append(c.Argv, string(c.QueryBuf[pos:c.BulkLen]))
					pos += int(c.BulkLen + 2)
				}
				c.BulkLen = -1
				c.MultiBulkLen--
			}
		}
	}

	if pos > 0 {
		// trim to pos
		c.QueryBuf = c.QueryBuf[pos:]
	}
	if c.MultiBulkLen == 0 {
		return C_OK
	}
	return C_ERR
}

func (s *Server) PauseClients(end time.Duration) {
	if !s.ClientsPaused || end > s.ClientsPauseEndTime {
		s.ClientsPauseEndTime = end
	}
	s.ClientsPaused = true
}

func (s *Server) ClientsArePasued() bool {
	if s.ClientsPaused && s.ClientsPauseEndTime < s.UnixTime {
		s.ClientsPaused = false
		iter := s.Clients.ListGetIterator(ITERATION_DIRECTION_INORDER)
		for node := iter.ListNext(); node != nil; node = iter.ListNext() {
			c := node.Value.(*Client)
			if c.WithFlags(CLIENT_SLAVE | CLIENT_BLOCKED) {
				continue
			}
			c.AddFlags(CLIENT_UNBLOCKED)
			s.ClientsUnblocked.ListAddNodeTail(c)
		}
	}
	return s.ClientsPaused
}

/* This function is called every time, in the client structure 'c', there is
 * more query buffer to process, because we read more data from the socket
 * or because a client was blocked and later reactivated, so there could be
 * pending query buffer, already representing a full command, to process. */
func (s *Server) ProcessInputBuffer(c *Client) {
	s.CurrentClient = c
	for len(c.QueryBuf) != 0 {
		if !c.WithFlags(CLIENT_SLAVE) && s.ClientsArePasued() {
			break
		}
		if c.WithFlags(CLIENT_CLOSE_AFTER_REPLY | CLIENT_CLOSE_ASAP) {
			break
		}
		if c.RequestType == 0 {
			if c.QueryBuf[0] == '*' {
				c.RequestType = PROTO_REQ_MULTIBULK
			} else {
				c.RequestType = PROTO_REQ_INLINE
			}
		}
		if c.RequestType == PROTO_REQ_INLINE {
			if s.ProcessInlineBuffer(c) != C_OK {
				break
			}
		} else if c.RequestType == PROTO_REQ_MULTIBULK {
			if s.ProcessMultiBulkBuffer(c) != C_ERR {
				break
			}
		} else {
			panic("Unknown request type")
		}

		if c.Argc == 0 {
			c.Reset()
		} else {
			if s.ProcessCommand(c) == C_OK {
				if c.WithFlags(CLIENT_MASTER) && !c.WithFlags(CLIENT_MULTI) {
					/* Update the applied replication offset of our master. */
					c.ReplyOff = c.ReadReplyOff - int64(len(c.QueryBuf))
				}
				if !c.WithFlags(CLIENT_BLOCKED) || c.BType != BLOCKED_MODULE {
					c.Reset()
				}
			}
			if s.CurrentClient == nil {
				break
			}
		}
	}
	s.CurrentClient = nil
}

/* If this function gets called we already read a whole
 * command, arguments are in the client argv/argc fields.
 * processCommand() execute the command or prepare the
 * server for a bulk read from the client.
 *
 * If C_OK is returned the client is still alive and valid and
 * other operations can be performed by the caller. Otherwise
 * if C_ERR is returned the client was destroyed (i.e. after QUIT). */
func (s *Server) ProcessCommand(c *Client) int64 {
	if c.Argv[0] == "quit" {
		/* The QUIT command is handled separately. Normal command procs will
		 * go through checking for replication and QUIT will cause trouble
		 * when FORCE_REPLICATION is enabled and would be implemented in
		 * a regular command proc. */
		s.AddReply(c, s.Shared.Ok)
		c.AddFlags(CLIENT_CLOSE_AFTER_REPLY)
		return C_ERR
	}
	c.Cmd = s.LookUpCommand(c.Argv[0])
	c.LastCmd = c.Cmd
	if c.Cmd == nil {
		c.FlagTransaction()
		s.AddReplyError(c, fmt.Sprintf("unknown command '%s'", c.Argv[0]))
		return C_OK
	}
	if (c.Cmd.Arity > 0 && c.Cmd.Arity != c.Argc) || c.Argc < -c.Cmd.Arity {
		c.FlagTransaction()
		s.AddReplyError(c, fmt.Sprintf("wrong number of arguments for '%s' command", c.Argv[0]))
		return C_OK
	}
	if (s.RequirePassword!=nil && c.Authenticated == 0) {
		c.Cmd.Process = AuthCommand
		if c.Cmd.Process AuthCommand {

		}
	}








	return C_OK
}

func (s *Server) LookUpCommand(name string) *Command{
	return s.Commands[name]
}

func (s *Server) SetProtocolError(err string, c *Client, pos int64) {
	if s.Verbosity <= LL_VERBOSE {
		//clientStr := c.CatClientInfoString(s)
		errorStr := fmt.Sprintf("Query buffer during protocol error: '%s'", c.QueryBuf)
		buf := make([]byte, len(errorStr))
		for i := 0; i < len(errorStr); i++ {
			if strconv.IsPrint(rune(errorStr[i])) {
				buf[i] = errorStr[i]
			} else {
				buf[i] = '.'
			}
		}
		c.AddFlags(CLIENT_CLOSE_AFTER_REPLY)
		c.QueryBuf = c.QueryBuf[pos:]
	}
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
