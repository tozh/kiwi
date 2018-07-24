package server
import ."redigo/src/structure"

type RedisObject struct {
	ObjType int
	Encoding int
	Lru uint
	RefCount int
	Ptr interface{}
}

type RedisDb struct {
	Dict map[string]interface{}
	Expires map[*RedisObject]uint
	Id uint
	AvgTTL uint
	DefragLater *List
}

type RedisCommand struct {
	Name string
	Arity int
	Flags int
	/* What keys should be loaded in background when calling this command? */
	FirstKey int
	LastKey int
	KeyStep int
	Msec uint
	Calls uint
}

type Client struct {
	Id int
	Fd int
	Db *RedisDb
	Name string
	QueryBuf string // buffer use to accumulate client query
	QueryBufPeak int
	Argc int  // count of arguments
	Argv []*RedisObject // arguments of current command
	Cmd *RedisCommand
	LastCmd *RedisObject
	Reply *List
	ReplySize int
	SentSize int // Amount of bytes already sent in the current buffer or object being sent.
	CreateTime uint
	LastInteraction uint
}

type RedisOp struct {
	Argc int  // count of arguments
	Argv []*RedisObject // arguments of current command
	DbId int
	Target int
	Cmd *RedisCommand
}

type RDBSaveInfo struct {
	ReplStreamDb int
	ReplIdIsSet bool
	ReplId string
	ReplOffset int
}


type RedisServer struct {
	Pid int
	PidFile string
	ConfigFile string
	ExecFile string
	ExecArgv []string

	RedisDb RedisDb

	Commands map[interface{}]RedisCommand
	OrigCommands map[interface{}]RedisCommand

	LruClock uint // Clock for LRU eviction
	ShutdownNeedAsap bool

	CronLoops int

	LoadModuleQueue *List  // List of modules to load at startup.


	// Network
	Port int  // TCP listening port
	BindAddr []string
	UnixSocket string  // UNIX socket path
	IpFileDesc []string  // TCP socket file descriptors
	SocketFileDesc string // Unix socket file descriptor
	Clients *List  // List of active clients
	//clientsToClose *List  // Clients to close asynchronously

	Loading bool  // Server is loading date from disk if true
	LoadingTotalSize int
	LoadingLoadedSize int
	LoadingStartTime uint


	// Configuration
	MaxIdleTimeSec uint  // Client timeout in seconds
	TcpKeepAlive bool


	// AOF persistence
	AofState int  //  AOF(ON|OFF|WAIT_REWRITE)
	AofChildPid int  // PID if rewriting process
	AofFileSync int  // Kind of fsync() policy
	AofFileName string
	AofRewirtePercent int  // Rewrite AOF if % growth is > M and...
	AofRewriteMinSize int
	AofRewriteMaxSize int
	AofRewriteScheduled bool  // Rewrite once BGSAVE terminates.
	AofRewriteBufBlocks *List  // Hold changes during AOF rewrites
	AofBuf string  // Aof buffer
	AofFileDesc int  // File descriptor of currently selected AOF file
	AofSelectedDb bool  // Currently selected DB in AOF
	AofPropagate []*RedisOp  // Additional command to propagate.


	// RDB persistence



	// Logging
	LogFile string
	SysLogEnable bool


	// Zip structure config
	HashMaxZiplistEntries int
	HashMaxZiplistValue int
	SetMaxInsetEntrie int
	ZSetMaxZiplistEntries int
	ZSetMaxZiplistvalue int
	





}