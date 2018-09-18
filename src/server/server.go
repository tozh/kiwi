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
	"strings"
	"sync/atomic"
	"net"
)

type accepted struct {
	conn net.Conn
	err  error
}

type Server struct {
	Pid                  int
	PidFile              string
	ConfigFile           string
	ExecFile             string
	ExecArgv             []string
	Hz                   int // serverCron() calls frequency in hertz
	Dbs                  [DEFAULT_DB_NUM]*Db
	DbNum                int
	Commands             map[string]*Command
	OrigCommands         map[string]*Command
	UnixTime             time.Time // UnixTime in nanosecond
	LruClock             time.Time // Clock for LRU eviction
	CronLoopCount        int
	NextClientId         int64
	Port                 int // TCP listening port
	BindAddrs            []string
	BindAddrCount        int       // Number of addresses in test_server.bindaddr[]
	UnixSocketPath       string    // UNIX socket path
	Clients              *SyncList // List of active clients
	ClientsMap           map[int64]*Client
	ClientMaxQueryBufLen int
	ClientMaxReplyBufLen int
	MaxClients           int
	ProtectedMode        bool // Don't accept external connections.
	RequirePassword      *string
	TcpKeepAlive         bool
	ProtoMaxBulkLen      int
	ClientMaxIdleTime    time.Duration
	Dirty                int64 // Changes to DB from the last save
	Shared               *Shared
	StatRejectedConn     int64
	StatConnCount        int64
	StatNetOutputBytes   int64
	StatNetInputBytes    int64
	StatNumCommands      int64
	ConfigFlushAll       bool
	MaxMemory            int
	Loading              bool
	LogLevel             int
	CloseCh              chan struct{}
	mutex                sync.RWMutex
	wg                   sync.WaitGroup
	events               *Events
	reusePort	bool
	numLoops int
}

var kiwiS *Server



func LruClock() time.Time {
	if 1000/kiwiS.Hz <= LRU_CLOCK_RESOLUTION {
		return kiwiS.LruClock
	} else {
		return GetLruClock()
	}
}

func GetLruClock() time.Time {
	return time.Now()
}

func UpdateCachedTime() {
	kiwiS.UnixTime = time.Now()
}

func UpdateLRUClock() {
	kiwiS.LruClock = time.Now()
}

func ServerCronHandler() {
	kiwiS.mutex.Lock()
	defer kiwiS.mutex.Unlock()
	kiwiS.wg.Add(1)
	defer kiwiS.wg.Done()
	UpdateCachedTime()
	UpdateLRUClock()
	kiwiS.CronLoopCount++
}

func ServerCron() {
	kiwiS.wg.Add(1)
	defer kiwiS.wg.Done()
	for {
		select {
		case <-kiwiS.CloseCh:
			kiwiS.ServerLogDebugF("-->%v\n", "ServerCron ------ SHUTDOWN")
			return
		case <-time.After(time.Millisecond * time.Duration(1000/kiwiS.Hz)):
			go ServerCronHandler()
		}
	}
}

func Call(c *Client) {
	// fmt.Println("Call")
	c.Cmd.Process(c)
	atomic.AddInt64(&kiwiS.StatNumCommands, 1)
}

func ProcessCommand(c *Client) int {
	// fmt.Println("ProcessCommand")
	cmdName := strings.ToLower(c.Argv[0])
	// fmt.Println([]byte(cmdName))
	c.Cmd = LookUpCommand(cmdName)
	if c.Cmd == nil {
		// fmt.Println("c.Cmd == nil")
		AddReplyError(c, fmt.Sprintf("unknown command '%kiwiS'", cmdName))
		return C_OK
	}
	if (c.Cmd.Arity > 0 && c.Cmd.Arity != c.Argc) || c.Argc < -c.Cmd.Arity {
		AddReplyError(c, fmt.Sprintf("wrong number of arguments for '%kiwiS' command", cmdName))
		return C_OK
	}
	if kiwiS.RequirePassword != nil && c.Authenticated == 0 && &c.Cmd.Process != &AuthCommand {
		// fmt.Println("Authenticated")
		AddReplyError(c, kiwiS.Shared.NoAuthErr)
		return C_OK
	}
	Call(c)
	return C_OK
}

func LookUpCommand(name string) *Command {
	return kiwiS.Commands[name]
}

func ProcessInline(c *Client) int {
	// fmt.Println("ProcessInline")

	// Search for end of line
	queryBuf := c.InBuf.Bytes()
	size := len(queryBuf)
	newline := IndexOfBytes(queryBuf, 0, size, '\n')
	if newline == -1 {
		if size > kiwiS.ClientMaxQueryBufLen {
			AddReplyError(c, "Protocol error: too big inline request")
			//SetProtocolError(c, "too big inline request", 0)
		}
		return C_ERR
	}
	if newline != 0 && newline != size && queryBuf[newline-1] == '\r' {
		// Handle the \r\n case.
		newline--
	}
	/* Split the input buffer up to the \r\n */
	argvs := SplitArgs(queryBuf[0:newline])
	if argvs == nil {
		AddReplyError(c, "Protocol error: unbalanced quotes in request")
		//SetProtocolError(c, "unbalanced quotes in inline request", 0)
		return C_ERR
	}

	// Leave data after the first line of the query in the buffer
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

func ProcessMultiBulk(c *Client) int {
	if c.Argc != 0 {
		panic("c.Argc != 0")
	}
	if c.MultiBulkLen == 0 {
		star, err := c.InBuf.ReadByte()
		if err != nil || star != '*' {
			AddReplyError(c, fmt.Sprintf("Protocol error: expected '*', got '%c'", star))
			//SetProtocolError(c, "expected $ but got something else", 0)
			return C_ERR
		}
		bulkNumStr, err := c.InBuf.ReadStringExclude('\r')
		if err != nil {
			return C_ERR
		}

		bulkNum, err := strconv.Atoi(bulkNumStr)
		if err != nil || bulkNum > 1024*1024 {
			AddReplyError(c, "Protocol error: invalid multibulk length")
			//SetProtocolError(c, "invalid multibulk length", 0)
			return C_ERR
		}
		if bulkNum <= 0 {
			return C_OK
		}
		c.InBuf.ReadByte() // pass the \n
		c.MultiBulkLen = bulkNum
		c.Argv = make([]string, c.MultiBulkLen)
	}
	if c.MultiBulkLen < 0 {
		return C_ERR
	}
	for c.MultiBulkLen > 0 {
		// Read bulk length if unknown
		dollar, err := c.InBuf.ReadByte()
		if err != nil || dollar != '$' {
			AddReplyError(c, fmt.Sprintf("Protocol error: expected '$', got '%c'", dollar))
			return C_ERR
		}
		bulkLenStr, err := c.InBuf.ReadStringExclude('\r')
		if err != nil {
			AddReplyError(c, fmt.Sprintf("Protocol error: invalid bulk length"))
			return C_ERR
		}
		bulkLen, err := strconv.Atoi(bulkLenStr)
		if err != nil || bulkLen > kiwiS.ProtoMaxBulkLen {
			AddReplyError(c, "Protocol error: invalid bulk length")
			return C_ERR
		}
		c.InBuf.ReadByte() // pass the \n

		bulk := c.InBuf.Next(bulkLen)
		if len(bulk) != bulkLen {
			AddReplyError(c, "Protocol error: invalid bulk format")
			return C_ERR
		}
		cr, _ := c.InBuf.ReadByte()
		lf, _ := c.InBuf.ReadByte()
		if cr != '\r' || lf != '\n' {
			AddReplyError(c, "Protocol error: invalid bulk format")
			return C_ERR
		}
		c.Argv[len(c.Argv)-c.MultiBulkLen] = string(bulk)
		c.Argc++
		c.MultiBulkLen--
	}
	if c.MultiBulkLen == 0 {
		return C_OK
	}
	return C_ERR
}

func ProcessInput(c *Client) {
	if c.RequestType == 0 {
		firstByte, _ := c.InBuf.ReadByteNotGoForward()
		if firstByte == '*' {
			c.RequestType = PROTO_REQ_MULTIBULK
		} else {
			c.RequestType = PROTO_REQ_INLINE
		}
	}
	if c.RequestType == PROTO_REQ_INLINE {
		if ProcessInline(c) != C_OK {
		}
	} else if c.RequestType == PROTO_REQ_MULTIBULK {
		if ProcessMultiBulk(c) != C_OK {

		}
	} else {
		panic("Unknown request type")
	}

	if c.Argc != 0 {
		ProcessCommand(c)
	}
}

// Write data in output buffers to client.
//func WriteToClient(c *Client) {
//	c.ReplyWriter.WriteByte(0)
//	atomic.AddInt64(&kiwiS.StatNetOutputBytes, 1)
//	if c.ReplyWriter.Flush() == nil {
//		c.SetLastInteraction(kiwiS.UnixTime)
//	}
//}
//
//func ProcessInput(c *Client) {
//	ProcessInput(c)
//	WriteToClient(c)
//	c.Reset()
//}

//func ReadQuery(c *Client, queryFinish chan int) {
//	// wait write send the signal
//	c.QueryCount++
//	reader := bufio.NewReaderSize(c.Conn, PROTO_IOBUF_LEN)
//	for {
//		recieved, err := reader.ReadBytes(0)
//		if err == io.EOF { // client side closed connection
//			BroadcastCloseClient(c)
//			return
//		}
//		if len(recieved) > 0 {
//			c.InBuf.Write(recieved)
//		}
//		if err == nil {
//			break
//		}
//	}
//	c.SetLastInteraction(kiwiS.UnixTime)
//	atomic.AddInt64(&kiwiS.StatNetInputBytes, int64(c.InBuf.Len()))
//	ProcessInput(c)
//	close(queryFinish)
//}

//func ProcessQueryLoop(c *Client) {
//	kiwiS.wg.Add(1)
//	defer kiwiS.wg.Done()
//	for {
//		queryFinish := make(chan int, 1)
//		go ReadQuery(c, queryFinish)
//
//		select {
//		case <-c.CloseCh:
//			// server closed, broadcast
//			close(queryFinish)
//			return
//		case <-queryFinish:
//			// query processing finished
//		}
//	}
//}

func ServerExists() (int, error) {
	// fmt.Println(os.TempDir() + "kiwi.pid")
	if kiwiPidFile, err1 := os.Open(os.TempDir() + "kiwi.pid"); err1 == nil {
		defer kiwiPidFile.Close()
		if pidStr, err2 := ioutil.ReadAll(kiwiPidFile); err2 == nil {
			if pid, err3 := strconv.Atoi(string(pidStr)); err3 == nil {
				if _, err4 := os.FindProcess(pid); err4 == nil {
					return pid, errors.New(fmt.Sprintf("Error! Wiki server is now runing. Pid is %d", pid))
				}
			}
		}
	}
	return 0, nil
}

//func CreateServer() *Server {
//	// fmt.Println("CreateServer")
//	pidFile := os.TempDir() + "kiwi.pid"
//	unixSocketPath := os.TempDir() + "kiwi.sock"
//	if pid, err1 := ServerExists(); err1 == nil {
//		pid = os.Getpid()
//		if kiwiPidFile, err2 := os.Create(pidFile); err2 == nil {
//			kiwiPidFile.WriteString(fmt.Sprintf("%d", pid))
//			kiwiPidFile.Close()
//		}
//
//		configPath, _ := filepath.Abs(filepath.Dir(os.Args[0]))
//		nowTime := time.Now()
//		kiwiS := Server{
//			Pid:                  pid,
//			PidFile:              pidFile,
//			ConfigFile:           configPath,
//			ExecFile:             os.Args[0],
//			ExecArgv:             os.Args,
//			Hz:                   10,
//			Dbs:                  make([]*Db, DEFAULT_DB_NUM),
//			DbNum:                DEFAULT_DB_NUM,
//			Commands:             make(map[string]*Command),
//			OrigCommands:         make(map[string]*Command),
//			UnixTime:             nowTime,
//			LruClock:             nowTime,
//			CronLoopCount:        0,
//			NextClientId:         0,
//			Port:                 9988,
//			BindAddrs:            make([]string, CONFIG_BINDADDR_MAX),
//			BindAddrCount:        0,
//			UnixSocketPath:       unixSocketPath,
//			CurrentClient:        nil,
//			Clients:              nil,
//			ClientsMap:           make(map[int]*Client),
//			ClientMaxQueryBufLen: PROTO_INLINE_MAX_SIZE,
//			MaxClients:           CONFIG_DEFAULT_MAX_CLIENTS,
//			ProtectedMode:        true,
//			RequirePassword:      nil,
//			TcpKeepAlive:         true,
//			ProtoMaxBulkLen:      CONFIG_DEFAULT_PROTO_MAX_BULK_LEN,
//			ClientMaxIdleTime:    5 * time.Second,
//			Dirty:                0,
//			Shared:               nil,
//			StatRejectedConn:     0,
//			StatConnCount:        0,
//			StatNetOutputBytes:   0,
//			StatNetInputBytes:    0,
//			StatNumCommands:      0,
//			ConfigFlushAll:       false,
//			MaxMemory:            CONFIG_DEFAULT_MAXMEMORY,
//			Loading:              false,
//			LogLevel:             LL_DEBUG,
//			CloseCh:              make(chan struct{}, 1),
//			mutex:                sync.RWMutex{},
//			wg:                   sync.WaitGroup{},
//		}
//		for i := 0; i < kiwiS.DbNum; i++ {
//			kiwiS.Dbs = append(kiwiS.DbCreateDb(i))
//		}
//		kiwiS.Clients = CreateSyncList()
//		kiwiS.BindAddrs = append(kiwiS.BindAddr"0.0.0.0")
//		kiwiS.BindAddrCount++
//		// // fmt.Println()
//		PopulateCommandTable()
//		return &kiwiS
//	} else {
//		// fmt.Println(err1)
//	}
//	os.Exit(1)
//	return nil
//}

func CreateServer() *Server {
	// fmt.Println("CreateServer")
	pidFile := os.TempDir() + "kiwi.pid"
	unixSocketPath := os.TempDir() + "kiwi.sock"
	pid := os.Getpid()
	fmt.Println("Pid", pid)

	configPath, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	nowTime := time.Now()
	s := Server{
		Pid:                  pid,
		PidFile:              pidFile,
		ConfigFile:           configPath,
		ExecFile:             os.Args[0],
		ExecArgv:             os.Args,
		Hz:                   10,
		Dbs:                  [DEFAULT_DB_NUM]*Db{},
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
	for i := 0; i < s.DbNum; i++ {
		s.Dbs[i] = CreateDb(i)
	}
	s.Clients = CreateSyncList()
	s.BindAddrs = append(s.BindAddrs, "0.0.0.0")
	s.BindAddrCount++
	// // fmt.Println()
	CreateShared()
	PopulateCommandTable()
	return &s

	//if pid, err1 := ServerExists(); err1 == nil {
	//	pid = os.Getpid()
	//	if kiwiPidFile, err2 := os.Create(pidFile); err2 == nil {
	//		kiwiPidFile.WriteString(fmt.Sprintf("%d", pid))
	//		kiwiPidFile.Close()
	//	}
	//
	//} else {
	//	// fmt.Println(err1)
	//}
	os.Exit(1)
	return nil
}

func StartServer() {
	// fmt.Println("StartServer")
	if kiwiS == nil {
		return
	}
	for _, addr := range kiwiS.BindAddrs {
		if addr != "" {
			go TcpServer(addr)
		}
	}
	//go UnixServer()
	go ServerCron()
	go CloseServerListener()
}

func HandleSignal() {
	// fmt.Println("HandleSignal")
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	kiwiS.ServerLogDebugF("-->%v: <%v>\n", "Signal", <-c)
	BroadcastCloseServer()
	kiwiS.wg.Wait()
	os.Exit(0)
}

func CloseServerListener() {
	// fmt.Println("CloseServerListener")
	kiwiS.wg.Add(1)
	defer kiwiS.wg.Done()
	select {
	case <-kiwiS.CloseCh:
		// fmt.Println("CloseServerListener ----> Close Server")
		CloseServer()
	}
}

func CloseServer() {
	// fmt.Println("CloseServer")
	// clear clients
	iter := kiwiS.Clients.ListGetIterator(ITERATION_DIRECTION_INORDER)
	for node := iter.ListNext(); node != nil; node = iter.ListNext() {
		BroadcastCloseClient(node.Value.(*Client))
	}
	defer os.Remove(kiwiS.UnixSocketPath)
	defer os.Remove(kiwiS.PidFile)
}
