package server

import (
	. "redigo/src/structure"
	. "redigo/src/constant"
	. "redigo/src/networking"
	"sync"
	"fmt"
	"strconv"
	"bytes"
	"net"
	"time"
	"os"
	"os/signal"
	"syscall"
	"io/ioutil"
	"errors"
	"path/filepath"
)

type Server struct {
	Pid                  int64
	PidFile              string
	ConfigFile           string
	ExecFile             string
	ExecArgv             []string
	Hz                   int64 // serverCron() calls frequency in hertz
	Dbs                  []*Db
	DbNum                int64
	Commands             map[string]*Command
	OrigCommands         map[string]*Command
	UnixTime             time.Time // UnixTime in nanosecond
	LruClock             time.Time // Clock for LRU eviction
	CronLoops            int64
	NextClientId         int64
	Port                 int64 // TCP listening port
	BindAddrs            []string
	BindAddrCount        int64  // Number of addresses in server.bindaddr[]
	UnixSocketPath       string // UNIX socket path
	CurrentClient        *Client
	Clients              *List // List of active clients
	ClientsMap           map[int64]*Client
	ClientsToClose       *List // Clients to close asynchronously
	ClientMaxQueryBufLen int64
	MaxClients           int64
	ProtectedMode        bool // Don't accept external connections.
	RequirePassword      *string
	TcpKeepAlive         bool
	ProtoMaxBulkLen      int64
	ClientsPaused        bool
	ClientsPauseEndTime  time.Time
	Dirty                int64 // Changes to DB from the last save
	Shared               *SharedObjects
	StatRejectedConn     int64
	StatConnCount        int64
	StatNetOutputBytes   int64
	StatNetInputBytes    int64
	StatNumCommands      int64
	ConfigFlushAll       bool
	mutex                sync.RWMutex
	MaxMemory            int64
	Loading              bool
	CloseCh              chan struct{}
	LogLevel             int64
	MaxIdleTime          time.Duration
}

func LruClock(s *Server) time.Time {
	if 1000/s.Hz <= LRU_CLOCK_RESOLUTION {
		// s.Hz >= 1, serverCron will update LRU, save resources
		// s.Hz default is 10
		return s.LruClock
	} else {
		return GetLruClock()
	}
}

//
//func GetLruClock() time.Time {
//  //int version time, speed should compared with time.Time
//	mstime := time.Now().UnixNano() / 1000
//	return mstime / LRU_CLOCK_RESOLUTION & LRU_CLOCK_MAX
//}
func GetLruClock() time.Time {
	return time.Now()
}

func CreateClient(s *Server, conn net.Conn) *Client {
	if conn != nil {
		if s.TcpKeepAlive {
			AnetSetTcpKeepALive(conn.(*net.TCPConn), s.TcpKeepAlive)
		}
	}
	createTime := s.UnixTime
	c := Client{
		Id:              0,
		Conn:            conn,
		Name:            "",
		QueryBuf:        make([]byte, PROTO_INLINE_MAX_SIZE),
		QueryBufPeak:    0,
		Argc:            0,                 // count of arguments
		Argv:            make([]string, 0), // arguments of current command
		Cmd:             nil,
		LastCmd:         nil,
		Reply:           ListCreate(),
		ReplySize:       0,
		CreateTime:      createTime,
		LastInteraction: createTime,
		Buf:             make([]byte, PROTO_REPLY_CHUNK_BYTES),
		BufPos:          0,
		SentLen:         0,
		Flags:           0,
		Node:            nil,
		PeerId:          "",
		RequestType:     0,
		MultiBulkLen:    0,
		BulkLen:         0,
		Authenticated:   0,
		ReadCh:          make(chan struct{}, 1),
		WriteCh:         make(chan struct{}, 1),
		CloseCh:         make(chan struct{}, 1),
		mutex:           sync.RWMutex{},
	}
	c.GetNextClientId(s)
	SelectDB(s, &c, 0)
	LinkClient(s, &c)
	return &c
}

func LinkClient(s *Server, c *Client) {
	s.Clients.ListAddNodeTail(c)
	s.ClientsMap[c.Id] = c
	c.Node = s.Clients.ListTail()
	s.StatConnCount++
}

func UnLinkClient(s *Server, c *Client) {
	if s.CurrentClient == c {
		s.CurrentClient = nil
	}
	if c.Conn != nil {
		s.Clients.ListDelNode(c.Node)
		c.Node = nil
		delete(s.ClientsMap, c.Id)
		s.StatConnCount--
		c.Conn.Close()
		c.Conn = nil
	}
}

func CloseClient(s *Server, c *Client) {
	fmt.Println("CloseClient")
	c.QueryBuf = nil
	c.Reply.ListEmpty()
	c.Reply = nil
	c.ResetArgv()
	UnLinkClient(s, c)
	if c.WithFlags(CLIENT_CLOSE_ASAP) {
		ln := s.ClientsToClose.ListSearchKey(c)
		s.ClientsToClose.ListDelNode(ln)
	}
	close(c.CloseCh)
	close(c.ReadCh)
	close(c.WriteCh)
}

func CloseClientAsync(s *Server, c *Client) {
	if c.WithFlags(CLIENT_CLOSE_ASAP) {
		return
	}
	c.AddFlags(CLIENT_CLOSE_ASAP)
	s.ClientsToClose.ListAddNodeTail(c)
}

func CloseClientsInAsyncList(s *Server) {
	s.ServerLogDebugF("-->%v\n", "CloseClientsInAsyncList")

	for s.ClientsToClose.ListLength() != 0 {
		ln := s.ClientsToClose.ListHead()
		c := ln.Value.(*Client)
		c.DeleteFlags(CLIENT_CLOSE_ASAP)
		CloseClient(s, c)
		s.ClientsToClose.ListDelNode(ln)
	}
}

func GetClientById(s *Server, id int64) *Client {
	return s.ClientsMap[id]
}

// Write data in output buffers to client.
func WriteToClient(s *Server, c *Client) {
	written := int64(0)
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
		c.LastInteraction = s.UnixTime
	}
	if !c.HasPendingReplies() {
		c.SentLen = 0
	}
}

/* Like processMultibulkBuffer(), but for the inline protocol instead of RESP,
 * this function consumes the client query buffer and creates a command ready
 * to be executed inside the client structure. Returns C_OK if the command
 * is ready to be executed, or C_ERR if there is still protocol to read to
 * have a well formed command. The function also returns C_ERR when there is
 * a protocol error: in such a case the client structure is setup to reply
 * with the error and close the connection. */
func ProcessInlineBuffer(s *Server, c *Client) int64 {
	// Search for end of line
	newline := bytes.IndexByte(c.QueryBuf, '\n')

	if newline == -1 {
		if len(c.QueryBuf) > PROTO_INLINE_MAX_SIZE {
			AddReplyError(s, c, "Protocol error: too big inline request")
			SetProtocolError(s, c, "too big inline request", 0)
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
		AddReplyError(s, c, "Protocol error: unbalanced quotes in request")
		SetProtocolError(s, c, "unbalanced quotes in inline request", 0)
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
func ProcessMultiBulkBuffer(s *Server, c *Client) int64 {
	pos := 0
	if c.MultiBulkLen == 0 {
		newline := bytes.IndexByte(c.QueryBuf, '\r')
		if newline < 0 {
			if len(c.QueryBuf) > PROTO_INLINE_MAX_SIZE {
				AddReplyError(s, c, "Protocol error: too big multibulk count request")
				SetProtocolError(s, c, "too big multibulk request", 0)
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
			AddReplyError(s, c, "Protocol error: invalid multibulk length")
			SetProtocolError(s, c, "invalid multibulk length", 0)
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
					AddReplyError(s, c, "Protocol error: too big bulk count string")
					SetProtocolError(s, c, "too big bulk count string", 0)
					return C_ERR
				}
				break
			}
			if len(c.QueryBuf)-newline < 2 {
				// end with \r\n, so \r cannot be the last char
				break
			}
			if c.QueryBuf[pos] != '$' {
				AddReplyError(s, c, fmt.Sprintf("Protocol error: expected '$', got '%c'", c.QueryBuf[pos]))
				SetProtocolError(s, c, "expected $ but got something else", 0)
				return C_ERR
			}
			nulkNum, err := strconv.Atoi(string(c.QueryBuf[pos+1 : newline]))
			if err != nil || int64(nulkNum) > s.ProtoMaxBulkLen {
				AddReplyError(s, c, "Protocol error: invalid bulk length")
				SetProtocolError(s, c, "invalid bulk length", 0)
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

func ReadQueryFromClient(s *Server, c *Client) {
	bufLen := int64(PROTO_IOBUF_LEN)
	queryLen := int64(len(c.QueryBuf))
	if c.RequestType == PROTO_REQ_MULTIBULK && c.MultiBulkLen != 0 && c.BulkLen != -1 && c.BulkLen >= PROTO_MBULK_BIG_ARG {
		remaining := c.BulkLen + 2 - int64(queryLen)
		if remaining < bufLen {
			bufLen = remaining
		}
	}
	if c.QueryBufPeak < queryLen {
		c.QueryBufPeak = queryLen
	}
	readCount, err := c.Conn.Read(c.QueryBuf)
	if err != nil {
		return
	}
	if readCount == 0 {
		return
	}
	c.LastInteraction = s.UnixTime
	//if c.WithFlags(CLIENT_MASTER) {
	//	c.ReadReplyOff += int64(readCount)
	//}
	s.StatNetInputBytes += int64(readCount)
	if int64(len(c.QueryBuf)) > s.ClientMaxQueryBufLen {
		CloseClient(s, c)
		return
	}
	ProcessInputBuffer(s, c)

}

/* This function is called every time, in the client structure 'c', there is
 * more query buffer to process, because we read more data from the socket
 * or because a client was blocked and later reactivated, so there could be
 * pending query buffer, already representing a full command, to process. */
func ProcessInputBuffer(s *Server, c *Client) {
	s.CurrentClient = c
	for len(c.QueryBuf) != 0 {
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
			if ProcessInlineBuffer(s, c) != C_OK {
				break
			}
		} else if c.RequestType == PROTO_REQ_MULTIBULK {
			if ProcessMultiBulkBuffer(s, c) != C_ERR {
				break
			}
		} else {
			panic("Unknown request type")
		}

		if c.Argc == 0 {
			c.Reset()
		} else {
			ProcessCommand(s, c)
			if s.CurrentClient == nil {
				break
			}
		}
	}
	s.CurrentClient = nil
}

/* Call() is the core of Redis execution of a command.
 *
 * The following flags can be passed:
 * CMD_CALL_NONE        No flags.
 * CMD_CALL_SLOWLOG     Check command speed and log in the slow log if needed.
 * CMD_CALL_STATS       Populate command stats.
 * CMD_CALL_PROPAGATE_AOF   Append command to AOF if it modified the dataset
 *                          or if the client flags are forcing propagation.
 * CMD_CALL_PROPAGATE_REPL  Send command to salves if it modified the dataset
 *                          or if the client flags are forcing propagation.
 * CMD_CALL_PROPAGATE   Alias for PROPAGATE_AOF|PROPAGATE_REPL.
 * CMD_CALL_FULL        Alias for SLOWLOG|STATS|PROPAGATE.
 *
 * The exact propagation behavior depends on the client flags.
 * Specifically:
 *
 * 1. If the client flags CLIENT_FORCE_AOF or CLIENT_FORCE_REPL are set
 *    and assuming the corresponding CMD_CALL_PROPAGATE_AOF/REPL is set
 *    in the call flags, then the command is propagated even if the
 *    dataset was not affected by the command.
 * 2. If the client flags CLIENT_PREVENT_REPL_PROP or CLIENT_PREVENT_AOF_PROP
 *    are set, the propagation into AOF or to slaves is not performed even
 *    if the command modified the dataset.
 *
 * Note that regardless of the client flags, if CMD_CALL_PROPAGATE_AOF
 * or CMD_CALL_PROPAGATE_REPL are not set, then respectively AOF or
 * slaves propagation will never occur.
 *
 * Client flags are modified by the implementation of a given command
 * using the following API:
 *
 * forceCommandPropagation(client *c, int flags);
 * preventCommandPropagation(client *c);
 * preventCommandAOF(client *c);
 * preventCommandReplication(client *c);
 *
 */
func Call(s *Server, c *Client, flags int64) {
	clientOldFlags := c.Flags
	c.DeleteFlags(CLIENT_FORCE_AOF | CLIENT_FORCE_REPL | CLIENT_PREVENT_PROP)

	dirty := s.Dirty
	start := time.Now()
	c.Cmd.Process(s, c)
	duration := time.Since(start)
	dirty = s.Dirty - dirty
	if dirty < 0 {
		dirty = 0
	}

	if flags&CMD_CALL_PROPAGATE != 0 {
		c.LastCmd.Duration = duration
		c.LastCmd.Calls++
		if c.Flags&CLIENT_PREVENT_AOF_PROP != CLIENT_PREVENT_PROP {
			propagateFlags := PROPAGATE_NONE
			if dirty != 0 {
				propagateFlags |= PROPAGATE_AOF | PROPAGATE_REPL
			}
			if c.WithFlags(CLIENT_FORCE_REPL) {
				propagateFlags |= PROPAGATE_REPL
			}
			if c.WithFlags(CLIENT_FORCE_AOF) {
				propagateFlags |= PROPAGATE_AOF
			}
			if c.WithFlags(CLIENT_PREVENT_REPL_PROP) {
				propagateFlags &= ^PROPAGATE_REPL
			}
			if c.WithFlags(CLIENT_PREVENT_AOF_PROP) {
				propagateFlags &= ^PROPAGATE_AOF
			}
			if propagateFlags != PROPAGATE_NONE && !c.Cmd.WithFlags(CMD_MODULE) {
				Propagate(s, c.Cmd, c.Db.Id, c.Argc, c.Argv, int64(propagateFlags))
			}
		}
	}
	c.DeleteFlags(CLIENT_FORCE_AOF | CLIENT_FORCE_REPL | CLIENT_PREVENT_PROP)
	c.AddFlags(clientOldFlags | CLIENT_FORCE_AOF | CLIENT_FORCE_REPL | CLIENT_PREVENT_PROP)
	s.StatNumCommands++
}

/* If this function gets called we already read a whole
 * command, arguments are in the client argv/argc fields.
 * processCommand() execute the command or prepare the
 * server for a bulk read from the client.
 *
 * If C_OK is returned the client is still alive and valid and
 * other operations can be performed by the caller. Otherwise
 * if C_ERR is returned the client was destroyed (i.e. after QUIT). */
func ProcessCommand(s *Server, c *Client) int64 {
	if c.Argv[0] == "quit" {
		/* The QUIT command is handled separately. Normal command procs will
		 * go through checking for replication and QUIT will cause trouble
		 * when FORCE_REPLICATION is enabled and would be implemented in
		 * a regular command proc. */
		AddReply(s, c, s.Shared.Ok)
		c.AddFlags(CLIENT_CLOSE_AFTER_REPLY)
		return C_ERR
	}
	c.Cmd = LookUpCommand(s, c.Argv[0])
	c.LastCmd = c.Cmd
	if c.Cmd == nil {
		AddReplyError(s, c, fmt.Sprintf("unknown command '%s'", c.Argv[0]))
		return C_OK
	}
	if (c.Cmd.Arity > 0 && c.Cmd.Arity != c.Argc) || c.Argc < -c.Cmd.Arity {
		AddReplyError(s, c, fmt.Sprintf("wrong number of arguments for '%s' command", c.Argv[0]))
		return C_OK
	}
	if s.RequirePassword != nil && c.Authenticated == 0 && &c.Cmd.Process != &AuthCommand {
		AddReplyError(s, c, s.Shared.NoAuthErr)
		return C_OK
	}
	/* Handle the maxmemory directive.
	 *
	 * First we try to free some memory if possible (if there are volatile
	 * keys in the dataset). If there are not the only thing we can do
	 * is returning an error. */
	if s.MaxMemory != 0 {
		result := FreeMemoryIfNeeded(s)
		if s.CurrentClient == nil {
			return C_ERR
		}
		if c.Cmd.WithFlags(CMD_DENYOOM) && result == C_ERR {
			AddReplyError(s, c, s.Shared.OOMErr)
			return C_OK
		}
	}

	/* Loading DB? Return an error if the command has not the
	 * CMD_LOADING flag. */
	if s.Loading && c.Cmd.WithFlags(CMD_LOADING) {
		AddReply(s, c, s.Shared.LoadingErr)
		return C_OK
	}

	Call(s, c, CMD_CALL_FULL)
	return C_OK
}

func FreeMemoryIfNeeded(s *Server) int64 {
	// TODO:
	return C_OK
}

func Propagate(s *Server, cmd *Command, dbid int64, argc int64, argv []string, flags int64) {
	//TODO:
}

func LookUpCommand(s *Server, name string) *Command {
	return s.Commands[name]
}

func SetProtocolError(s *Server, c *Client, err string, pos int64) {
	s.ServerLogErrorF("%s\n", err)
	if s.LogLevel <= LL_INFO {
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

func GetAllClientInfoString(s *Server, ctype int64) string {
	str := bytes.Buffer{}

	iter := s.Clients.ListGetIterator(ITERATION_DIRECTION_INORDER)
	for node := iter.ListNext(); node != nil; node = iter.ListNext() {
		c := node.Value.(*Client)
		if ctype != -1 && c.GetClientType() != ctype {
			continue
		}
		str.WriteString(CatClientInfoString(s, c))
		str.WriteByte('\n')
	}
	return str.String()
}

func DbDeleteSync(s *Server, c *Client, key string) bool {
	// TODO expire things
	c.Db.Delete(key)
	return true
}

func DbDeleteAsync(s *Server, c *Client, key string) bool {
	// TODO
	c.Db.Delete(key)
	return true
}

func SelectDB(s *Server, c *Client, dbId int64) int64 {
	if dbId < 0 || dbId >= s.DbNum {
		return C_ERR
	}
	c.Db = s.Dbs[dbId]
	return C_OK
}

func UpdateCachedTime(s *Server) {
	s.mutex.Lock()
	s.UnixTime = time.Now()
	s.mutex.Unlock()
}

func UpdateLRUClock(s *Server) {
	s.mutex.Lock()
	s.LruClock = time.Now()
	s.mutex.Unlock()
}

func ServerCron(s *Server) {
	for {
		select {
		case <-s.CloseCh:
			return
		default:
			s.ServerLogDebugF("-->%v, Loop: %d\n", "ServerCron", s.CronLoops)
			UpdateCachedTime(s)
			UpdateLRUClock(s)
			ClientCron(s)
			DbsCron(s)
			CloseClientsInAsyncList(s)
			s.CronLoops++
			time.Sleep(100 *time.Millisecond)
		}
	}
}

func ClientCronHandler(s *Server, c *Client, wg sync.WaitGroup) {
	// Client Time Out
	wg.Add(1)
	c.mutex.Lock()
	if s.MaxIdleTime != 0*time.Millisecond && time.Since(c.LastInteraction) > s.MaxIdleTime {
		// time out, no interaction time longer than the idle time for client
		CloseClient(s, c)
	}

	// Client Resize QueryBuf
	idleTime := s.UnixTime.Sub(c.LastInteraction)
	queryBufSize := int64(len(c.QueryBuf) + cap(c.QueryBuf))
	/* There are two conditions to resize the query buffer:
 * 1) Query buffer is > BIG_ARG and too big for latest peak.
 * 2) Query buffer is > BIG_ARG and client is idle. */
	if queryBufSize > PROTO_MBULK_BIG_ARG && ((queryBufSize/c.QueryBufPeak+1) > 2 || idleTime > 2 * time.Millisecond) {
		if cap(c.QueryBuf) > 1024 * 4 {
			newSlice := make([]byte, len(c.QueryBuf))
			copy(newSlice, c.QueryBuf)
			c.QueryBuf = newSlice
		}
	}
	c.QueryBufPeak = 0
	c.mutex.Unlock()
	wg.Done()
}
func ClientCron(s *Server) {
	wg := sync.WaitGroup{}
	iter := s.Clients.ListGetIterator(ITERATION_DIRECTION_INORDER)
	for node := iter.ListNext(); node != nil; node = iter.ListNext() {
		c := node.Value.(*Client)
		go ClientCronHandler(s, c, wg)
	}
	wg.Wait()
}

func DbsCron(s *Server) {

}

func ServerExists() (int, error) {
	fmt.Printf("-->%v\n", "ServerExists")

	if redigoPidFile, err1 := os.Open(os.TempDir() + "redigo.pid"); err1 == nil {
		defer redigoPidFile.Close()
		if pidStr, err2 := ioutil.ReadAll(redigoPidFile); err2 == nil {
			if pid, err3 := strconv.Atoi(string(pidStr)); err3 == nil {
				if _, err4 := os.FindProcess(pid); err4 == nil {
					return pid, errors.New(fmt.Sprintf("Error! Redigo server is now runing. Pid is %d", pid))
				}
			}
		}
	}
	return 0, nil
}

func CreateServer() *Server {
	fmt.Printf("-->%s\n", "CreateServer")
	pidFile := os.TempDir() + "redigo.pid"
	unixSocketPath := os.TempDir() + "redigo.sock"
	if pid, err1 := ServerExists(); err1 == nil {
		pid = os.Getpid()
		if redigoPidFile, err2 := os.Create(pidFile); err2 == nil {
			redigoPidFile.WriteString(fmt.Sprintf("%d", pid))
			redigoPidFile.Close()
		}

		configPath, _ := filepath.Abs(filepath.Dir(os.Args[0]))
		nowTime := time.Now()
		s := Server{
			int64(pid),
			pidFile,
			configPath,
			os.Args[0],
			os.Args,
			10,
			make([]*Db, DEFAULT_DB_NUM),
			DEFAULT_DB_NUM,
			make(map[string]*Command),
			make(map[string]*Command),
			nowTime,
			nowTime,
			0,
			0,
			9988,
			make([]string, CONFIG_BINDADDR_MAX),
			0,
			unixSocketPath,
			nil,
			nil,
			make(map[int64]*Client),
			nil,
			//make([]ClientBufferLimitsConfig, CLIENT_TYPE_OBUF_COUNT),
			0,
			CONFIG_DEFAULT_MAX_CLIENTS,
			true,
			nil,
			true,
			CONFIG_DEFAULT_PROTO_MAX_BULK_LEN,
			false,
			time.Unix(0, 0),
			0,
			nil,
			0,
			0,
			0,
			0,
			0,
			false,
			sync.RWMutex{},
			CONFIG_DEFAULT_MAXMEMORY,
			false,
			make(chan struct{}, 1),
			LL_DEBUG,
			0 * time.Millisecond,
		}
		for i := int64(0); i < s.DbNum; i++ {
			s.Dbs = append(s.Dbs, CreateDb(i))
		}
		s.Clients = ListCreate()
		s.ClientsToClose = ListCreate()
		s.BindAddrs = append(s.BindAddrs, "0.0.0.0")
		s.BindAddrCount++
		CreateShared(&s)
		return &s
	} else {
		fmt.Println(err1)
	}
	os.Exit(1)
	return nil
}

func StartServer(s *Server) {
	s.ServerLogDebugF("-->%v\n", "StartServer")
	if s == nil {
		return
	}
	for _, addr := range s.BindAddrs {
		if addr != "" {
			go TcpServer(s, addr)
		}
	}
	go UnixServer(s)
	go ServerCron(s)
}

func CloseServer(s *Server) {
	s.ServerLogDebugF("-->%v\n", "CloseServer")
	CloseClientsInAsyncList(s)

	// clear clients
	iter := s.Clients.ListGetIterator(ITERATION_DIRECTION_INORDER)
	for node := iter.ListNext(); node != nil; node = iter.ListNext() {
		CloseClient(s, node.Value.(*Client))
	}

	//notify server is closed
	close(s.CloseCh)
	os.Remove(s.PidFile)
}

func HandleSignal(s *Server) {
	s.ServerLogDebugF("-->%v\n", "HandleSignal")
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	s.ServerLogDebugF("-->%v: <%v>\n", "Signal", <-c)
	CloseServer(s)
}
