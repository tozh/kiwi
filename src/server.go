package src

type RedisObject struct {
	objType int
	encoding int
	lru uint
	refCount int
	ptr interface{}
}

type RedisDb struct {
	dict map[string]interface{}
	expires map[*RedisObject]uint
	id uint
	avgTTL uint
	defragLater *List
}

type RedisCommand struct {
	name string
	arity int
	flags int
	/* What keys should be loaded in background when calling this command? */
	firstKey int
	lastKey int
	keyStep int
	msec uint
	calls uint
}

type Client struct {
	id int
	fd int
	db *RedisDb
	name string
	queryBuf string // buffer use to accumulate client query
	queryBufPeak int
	argc int  // count of arguments
	argv []*RedisObject // arguments of current command
	cmd *RedisCommand
	lastCmd *RedisObject
	reply *List
	replySize int
	sentSize int // Amount of bytes already sent in the current buffer or object being sent.
	createTime uint
	lastInteraction uint
}

type RedisOp struct {
	argc int  // count of arguments
	argv []*RedisObject // arguments of current command
	dbId int
	target int
	cmd *RedisCommand
}

type RDBSaveInfo struct {
	replStreamDb int
	replIdIsSet bool
	replId string
	replOffset int
}


type RedisServer struct {
	pid int
	pidFile string
	configFile string
	execFile string
	execArgv []string

	redisDb RedisDb

	commands map[interface{}]RedisCommand
	origCommands map[interface{}]RedisCommand

	lruClock uint // Clock for LRU eviction
	shutdownNeedAsap bool

	cronLoops int

	loadModuleQueue *List  // List of modules to load at startup.


	// Network
	port int  // TCP listening port
	bindAddr []string
	unixSocket string  // UNIX socket path
	ipFileDesc []string  // TCP socket file descriptors
	socketFileDesc string // Unix socket file descriptor
	clients *List  // List of active clients
	//clientsToClose *List  // Clients to close asynchronously

	loading bool  // Server is loading date from disk if true
	loadingTotalSize int
	loadingLoadedSize int
	loadingStartTime uint


	// Configuration
	maxIdleTimeSec uint  // Client timeout in seconds
	tcpKeepAlive bool


	// AOF persistence
	aofState int  //  AOF(ON|OFF|WAIT_REWRITE)
	aofChildPid int  // PID if rewriting process
	aofFileSync int  // Kind of fsync() policy
	aofFileName string
	aofRewirtePercent int  // Rewrite AOF if % growth is > M and...
	aofRewriteMinSize int
	aofRewriteMaxSize int
	aofRewriteScheduled bool  // Rewrite once BGSAVE terminates.
	aofRewriteBufBlocks *List  // Hold changes during AOF rewrites
	aofBuf string  // aof buffer
	aofFileDesc int  // File descriptor of currently selected AOF file
	aofSelectedDb bool  // Currently selected DB in AOF
	aofPropagate []*RedisOp  // Additional command to propagate.


	// RDB persistence



	// Logging
	logFile string
	sysLogEnable bool


	// Zip structure config
	HashMaxZiplistEntries int
	HashMaxZiplistValue int
	SetMaxInsetEntrie int
	ZSetMaxZiplistEntries int
	ZSetMaxZiplistvalue int
	





}