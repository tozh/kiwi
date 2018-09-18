package server

import (
	"sync"
	"fmt"
	"strconv"
	"time"
	"os"
	"io/ioutil"
	"errors"
	"path/filepath"
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
	ClientsMap           map[int64]*KiwiClient
	ClientMaxQueryBufLen int
	ClientMaxReplyBufLen int
	MaxClients           int64
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
	events               Events
	reusePort            bool
	numLoops             int
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
//			ClientsMap:           make(map[int]*KiwiClient),
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

func InitServer() {
	// fmt.Println("CreateServer")
	pidFile := os.TempDir() + "kiwi.pid"
	unixSocketPath := os.TempDir() + "kiwi.sock"
	pid := os.Getpid()
	fmt.Println("Pid", pid)

	configPath, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	nowTime := time.Now()
	kiwiS = &Server{
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
		ClientsMap:           make(map[int64]*KiwiClient),
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
	for i := 0; i < kiwiS.DbNum; i++ {
		kiwiS.Dbs[i] = CreateDb(i)
	}
	kiwiS.Clients = CreateSyncList()
	kiwiS.BindAddrs = append(kiwiS.BindAddrs, "0.0.0.0")
	kiwiS.BindAddrCount++
	CreateShared()
	PopulateCommandTable()
	kiwiS.events = CreateKiwiServerEvents()
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
}

func StartServer() {
	// fmt.Println("StartServer")

	if kiwiS == nil {
		return
	}
	for _, addr := range kiwiS.BindAddrs {
		if addr != "" {
			go EventServe(kiwiS.events, "")
		}
	}
	//go UnixServer()
	//go ServerCron()

}

func GenerateAddrs() (addrs []string) {
	addrs = []string{}
	for _, addr := range kiwiS.BindAddrs {
		append(addrs, fmt.)
	}
}

func CloseServer() {
	// fmt.Println("CloseServer")
	defer os.Remove(kiwiS.UnixSocketPath)
	defer os.Remove(kiwiS.PidFile)
}

func BroadcastCloseServer() {
	// fmt.Println("BroadcastCloseServer")
	close(kiwiS.CloseCh)
}
