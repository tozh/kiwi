package server

import (
	. "redigo/src/structure"
	. "redigo/src/db"
	. "redigo/src/object"
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

type Client struct {
	Id int64
	Fd int64
	Db *Db
	Name string
	QueryBuf string // buffer use to accumulate client query
	QueryBufPeak int64
	Argc int64       // count of arguments
	Argv []string // arguments of current command
	Cmd *RedisCommand
	LastCmd *RedisCommand
	Reply *List
	ReplySize int64
	SentSize int64 // Amount of bytes already sent in the current buffer or object being sent.
	CreateTime int64
	LastInteraction  int64
}

type Op struct {
	Argc int64       // count of arguments
	Argv []string // arguments of current command
	DbId int64
	Target int64
	Cmd *RedisCommand
}

//type RDBSaveInfo struct {
//	ReplStreamDb int64
//	ReplIdIsSet bool
//	ReplId string
//	ReplOffset int64
//}


type Server struct {
	Pid int64
	PidFile string
	ConfigFile string
	ExecFile string
	ExecArgv []string
	Hz int64		// serverCron() calls frequency in hertz
	Db *Db

	Commands map[interface{}]RedisCommand
	OrigCommands map[interface{}]RedisCommand

	LruClock int64 // Clock for LRU eviction
	ShutdownNeedAsap bool

	CronLoops int64

	LoadModuleQueue *List  // List of modules to load at startup.


	// Network
	Port int64  // TCP listening port
	BindAddr []string
	UnixSocket string  // UNIX socket path
	IpFileDesc []string  // TCP socket file descriptors
	SocketFileDesc string // Unix socket file descriptor
	Clients *List  // List of active clients
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
}






