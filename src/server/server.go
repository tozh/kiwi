package server

import (
	"sync"
	"fmt"
	"strconv"
	"time"
	"os"
	"os/signal"
	"syscall"
	"io/ioutil"
	"errors"
	"path/filepath"
	"io"
	"bytes"
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
	CronLoopCount        int64
	NextClientId         int64
	Port                 int64 // TCP listening port
	BindAddrs            []string
	BindAddrCount        int64  // Number of addresses in test_server.bindaddr[]
	UnixSocketPath       string // UNIX socket path
	CurrentClient        *Client
	Clients              *SyncList // List of active clients
	ClientsMap           map[int64]*Client
	ClientMaxQueryBufLen int64
	ClientMaxReplyBufLen int64
	MaxClients           int64
	ProtectedMode        bool // Don't accept external connections.
	RequirePassword      *string
	TcpKeepAlive         bool
	ProtoMaxBulkLen      int64
	ClientMaxIdleTime    time.Duration
	Dirty                int64 // Changes to DB from the last save
	Shared               *SharedObjects
	StatRejectedConn     int64
	StatConnCount        int64
	StatNetOutputBytes   int64
	StatNetInputBytes    int64
	StatNumCommands      int64
	ConfigFlushAll       bool
	MaxMemory            int64
	Loading              bool
	LogLevel             int64
	CloseCh              chan struct{}
	mutex                sync.RWMutex
	wg                   sync.WaitGroup
}

func LruClock(s *Server) time.Time {
	if 1000/s.Hz <= LRU_CLOCK_RESOLUTION {
		return s.LruClock
	} else {
		return GetLruClock()
	}
}

func GetLruClock() time.Time {
	return time.Now()
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
	c.ReplyList.ListEmpty()
	c.ReplyList = nil
	c.ResetArgv()
	UnLinkClient(s, c)
	close(c.CloseCh)
}

func GetClientById(s *Server, id int64) *Client {
	return s.ClientsMap[id]
}

// Write data in output buffers to client.
func WriteToClient(s *Server, c *Client) {
	written := int64(0)
	for c.HasPendingReplies() {
		if c.ReplyBufSize > 0 {
			n, err := c.Write(c.ReplyBuf[:c.ReplyBufSize])
			if err == nil {
				if n <= 0 {
					break
				}
				c.SentLen += int64(n)
				written += n
			}
			if c.SentLen == c.ReplyBufSize {
				c.SentLen = 0
				c.ReplyBufSize = 0
			}
		} else {
			str := c.ReplyList.ListHead().Value.(*string)
			length := int64(len(*str))
			if length == 0 {
				c.ReplyList.ListDelNode(c.ReplyList.ListHead())
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
				c.ReplyList.ListDelNode(c.ReplyList.ListHead())
				c.SentLen = 0
				c.ReplyListSize -= length
				if c.ReplyList.ListLength() == 0 {
					c.ReplyListSize = 0
				}
			}
		}
		if written > NET_MAX_WRITES_PER_EVENT {
			break
		}
	}
	s.StatNetOutputBytes += written
	if written > 0 {
		c.SetLastInteraction(s.UnixTime)
	}
	if !c.HasPendingReplies() {
		c.SentLen = 0
	}
}

func ProcessInlineBuffer(s *Server, c *Client) int64 {
	// Search for end of line
	newline := IndexOfBytes(c.QueryBuf, 0, int(c.QueryBufSize), '\n')

	if newline == -1 {
		if c.QueryBufSize > s.ClientMaxQueryBufLen {
			AddReplyError(s, c, "Protocol error: too big inline request")
			SetProtocolError(s, c, "too big inline request", 0)
		}
		return C_ERR
	}
	if newline != 0 && newline != int(c.QueryBufSize) && c.QueryBuf[newline-1] == '\r' {
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
	//c.QueryBuf = c.QueryBuf[newline+2:]
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

func ProcessMultiBulkBuffer(s *Server, c *Client) int64 {
	pos := 0
	if c.Argc != 0 {
		panic("c.Argc != 0")
	}
	if c.MultiBulkLen == 0 {
		newline := IndexOfBytes(c.QueryBuf, 0, int(c.QueryBufSize), '\r')
		if newline < 0 {
			if c.QueryBufSize > s.ClientMaxQueryBufLen {
				AddReplyError(s, c, "Protocol error: too big multibulk count request")
				SetProtocolError(s, c, "too big multibulk request", 0)
			}
			return C_ERR
		}
		if c.QueryBufSize-int64(newline) < 2 {
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
			// null multibulk
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
			newline := IndexOfBytes(c.QueryBuf, pos, int(c.QueryBufSize), '\r')
			if newline < 0 {
				if c.QueryBufSize > s.ClientMaxQueryBufLen {
					AddReplyError(s, c, "Protocol error: too big bulk count string")
					SetProtocolError(s, c, "too big bulk count string", 0)
					return C_ERR
				}
				break
			}
			if c.QueryBufSize-int64(newline) < 2 {
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
			if c.QueryBufSize-int64(pos) < c.BulkLen+2 {
				break
			} else {
				c.Argv = append(c.Argv, string(c.QueryBuf[pos:c.BulkLen]))
				c.Argc++
				pos += int(c.BulkLen + 2)
				c.BulkLen = -1
				c.MultiBulkLen--
			}
		}
	}
	if c.MultiBulkLen == 0 {
		return C_OK
	}
	return C_ERR
}

func ReadFromClient(s *Server, c *Client, readCh chan int64) {
	n, err := c.Conn.Read(c.QueryBuf)
	if err != nil {

		if err == io.EOF {
			fmt.Println("ReadFromClient: EOF !!!!")
		} else {
			fmt.Println(err)
		}
		BroadcastCloseClient(c)
		readCh <- C_ERR
		return
	}
	c.ReadCount++
	if !c.WithFlags(CLIENT_LUA) && c.MaxIdleTime == 0 {
		c.HeartBeatCh <- c.ReadCount
	}
	c.QueryBufSize = int64(n)
	c.SetLastInteraction(s.UnixTime)
	s.mutex.Lock()
	s.StatNetInputBytes += c.QueryBufSize
	s.mutex.Unlock()
	if c.QueryBufSize > s.ClientMaxQueryBufLen {
		BroadcastCloseClient(c)
		readCh <- C_ERR
		return
	}
	ProcessInputBuffer(s, c)
	readCh <- C_OK
}

func ProcessInputBuffer(s *Server, c *Client) {
	s.CurrentClient = c
	for len(c.QueryBuf) != 0 {
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
			if ProcessMultiBulkBuffer(s, c) != C_OK {
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

func Call(s *Server, c *Client) {
	c.Cmd.Process(s, c)
	s.StatNumCommands++
}

func ProcessCommand(s *Server, c *Client) int64 {
	c.Cmd = LookUpCommand(s, c.Argv[0])
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
	Call(s, c)
	return C_OK
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
	s.UnixTime = time.Now()
}

func UpdateLRUClock(s *Server) {
	s.LruClock = time.Now()
}

func ServerCronHandler(s *Server) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.wg.Add(1)
	defer s.wg.Done()
	UpdateCachedTime(s)
	UpdateLRUClock(s)
	s.CronLoopCount++
}

func ServerCron(s *Server) {
	s.wg.Add(1)
	defer s.wg.Done()
	for {
		select {
		case <-s.CloseCh:
			s.ServerLogDebugF("-->%v\n", "ServerCron ------ SHUTDOWN")
			return
		case <-time.After(time.Millisecond * time.Duration(1000/s.Hz)):
			go ServerCronHandler(s)
		}
	}
}

func ServerExists() (int, error) {
	fmt.Printf("-->%v\n", "ServerExists")
	if redigoPidFile, err1 := os.Open(os.TempDir() + "redigo.pid"); err1 == nil {
		defer redigoPidFile.Close()
		if pidStr, err2 := ioutil.ReadAll(redigoPidFile); err2 == nil {
			if pid, err3 := strconv.Atoi(string(pidStr)); err3 == nil {
				if _, err4 := os.FindProcess(pid); err4 == nil {
					return pid, errors.New(fmt.Sprintf("Error! Redigo test_server is now runing. Pid is %d", pid))
				}
			}
		}
	}
	return 0, nil
}

func CreateServer() *Server {
	fmt.Println("CreateServer")
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
			Pid:                  int64(pid),
			PidFile:              pidFile,
			ConfigFile:           configPath,
			ExecFile:             os.Args[0],
			ExecArgv:             os.Args,
			Hz:                   10,
			Dbs:                  make([]*Db, DEFAULT_DB_NUM),
			DbNum:                DEFAULT_DB_NUM,
			Commands:             make(map[string]*Command),
			OrigCommands:         make(map[string]*Command),
			UnixTime:             nowTime,
			LruClock:             nowTime,
			CronLoopCount:        0,
			NextClientId:         0,
			Port:                 9988,
			BindAddrs:            make([]string, CONFIG_BINDADDR_MAX),
			BindAddrCount:        0,
			UnixSocketPath:       unixSocketPath,
			CurrentClient:        nil,
			Clients:              nil,
			ClientsMap:           make(map[int64]*Client),
			ClientMaxQueryBufLen: PROTO_INLINE_MAX_SIZE,
			MaxClients:           CONFIG_DEFAULT_MAX_CLIENTS,
			ProtectedMode:        true,
			RequirePassword:      nil,
			TcpKeepAlive:         true,
			ProtoMaxBulkLen:      CONFIG_DEFAULT_PROTO_MAX_BULK_LEN,
			ClientMaxIdleTime:    5 * time.Second,
			Dirty:                0,
			Shared:               nil,
			StatRejectedConn:     0,
			StatConnCount:        0,
			StatNetOutputBytes:   0,
			StatNetInputBytes:    0,
			StatNumCommands:      0,
			ConfigFlushAll:       false,
			MaxMemory:            CONFIG_DEFAULT_MAXMEMORY,
			Loading:              false,
			LogLevel:             LL_DEBUG,
			CloseCh:              make(chan struct{}, 1),
			mutex:                sync.RWMutex{},
			wg:                   sync.WaitGroup{},
		}
		for i := int64(0); i < s.DbNum; i++ {
			s.Dbs = append(s.Dbs, CreateDb(i))
		}
		s.Clients = CreateSyncList()
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
	fmt.Println("StartServer")
	if s == nil {
		return
	}
	for _, addr := range s.BindAddrs {
		if addr != "" {
			go TcpServer(s, addr)
		}
	}
	//go UnixServer(s)
	go ServerCron(s)
	go CloseServerListener(s)
}

func HandleSignal(s *Server) {
	fmt.Println( "HandleSignal")
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	s.ServerLogDebugF("-->%v: <%v>\n", "Signal", <-c)
	BroadcastCloseServer(s)
	s.wg.Wait()
	os.Exit(0)
}

func CloseServerListener(s *Server) {
	s.wg.Add(1)
	defer s.wg.Done()
	select {
	case <-s.CloseCh:
		fmt.Println("CloseServerListener ----> Close Server")
		CloseServer(s)
	}
}

func CloseServer(s *Server) {
	fmt.Println( "CloseServer")
	// clear clients
	iter := s.Clients.ListGetIterator(ITERATION_DIRECTION_INORDER)
	for node := iter.ListNext(); node != nil; node = iter.ListNext() {
		BroadcastCloseClient(node.Value.(*Client))
	}
	defer os.Remove(s.UnixSocketPath)
	defer os.Remove(s.PidFile)
}
