package server

/* constants for linkedlist */
const ITERATION_DIRECTION_INORDER = 1
const ITERATION_DIRECTION_REVERSE_ORDER = -1

/* constants for zskiplist */
const ZSKIPLIST_MAXLEVEL = 64
const ZSKIPLIST_P = 0.25
const ZSKIPLIST_RANDOM_MAXLEVEL = 0xFFFF * ZSKIPLIST_P

/* constants for Object */
const OBJ_ENCODING_STR = 0
const OBJ_ENCODING_INT = 1
const OBJ_ENCODING_HT = 2
const OBJ_ENCODING_ZIPMAP = 3
const OBJ_ENCODING_LINKEDLIST = 4
const OBJ_ENCODING_ZIPLIST = 5
const OBJ_ENCODING_INTSET = 6
const OBJ_ENCODING_SKIPLIST = 7
const OBJ_ENCODING_QUICKLIST = 8
const OBJ_ENCODING_STREAM = 9

const OBJ_RTYPE_STR = 0
const OBJ_RTYPE_INT = 1
const OBJ_RTYPE_LIST = 2
const OBJ_RTYPE_ZSET = 3
const OBJ_RTYPE_HASH = 4
const OBJ_RTYPE_SET = 5

const DICT_ON = 0
const DICT_ERR = 1

const LRU_BITS = 24
const LRU_CLOCK_MAX = (1 << LRU_BITS) - 1 /* Max value of obj->lru */
const LRU_CLOCK_RESOLUTION = 1000         /* LRU clock resolution in ms */

const OBJ_SET_NO_FLAGS = 0
const OBJ_SET_NX = 1 << 0 /* Set if key not exists. */
const OBJ_SET_XX = 1 << 1 /* Set if key exists. */
const OBJ_SET_EX = 1 << 2 /* Set if time in seconds is given */
const OBJ_SET_PX = 1 << 3 /* Set if time in ms in given */

const SHARED_INTEGERS = 10000
const SHARED_BULKHDR_LEN = 32

const C_OK = 0
const C_ERR = 1

// Client flags

/* Client flags */
const CLIENT_SLAVE = 1 << 0             /* This client is a slave test_server */
const CLIENT_MASTER = 1 << 1            /* This client is a master test_server */
const CLIENT_MONITOR = 1 << 2           /* This client is a slave monitor, see MONITOR */
const CLIENT_MULTI = 1 << 3             /* This client is in a MULTI context */
const CLIENT_BLOCKED = 1 << 4           /* The client is waiting in a blocking operation */
const CLIENT_DIRTY_CAS = 1 << 5         /* Watched keys modified. EXEC will fail. */
const CLIENT_CLOSE_AFTER_REPLY = 1 << 6 /* Close after writing entire reply. */
const CLIENT_UNBLOCKED = 1 << 7         /* This client was unblocked and is stored in
                                  test_server.unblocked_clients */
const CLIENT_LUA = 1 << 8                 /* This is a non connected client used by Lua */
const CLIENT_ASKING = 1 << 9              /* Client issued the ASKING command */
const CLIENT_CLOSE_ASAP = 1 << 10         /* Close this client ASAP */
const CLIENT_UNIX_SOCKET = 1 << 11        /* Client connected via Unix domain socket */
const CLIENT_DIRTY_EXEC = 1 << 12         /* EXEC will fail for errors while queueing */
const CLIENT_MASTER_FORCE_REPLY = 1 << 13 /* Queue replies even if is master */
const CLIENT_FORCE_AOF = 1 << 14          /* Force AOF propagation of current cmd. */
const CLIENT_FORCE_REPL = 1 << 15         /* Force replication of current cmd. */
const CLIENT_PRE_PSYNC = 1 << 16          /* Instance don't understand PSYNC. */
const CLIENT_READONLY = 1 << 17           /* Cluster client is in read-only state. */
const CLIENT_PUBSUB = 1 << 18             /* Client is in Pub/Sub mode. */
const CLIENT_PREVENT_AOF_PROP = 1 << 19   /* Don't propagate to AOF. */
const CLIENT_PREVENT_REPL_PROP = 1 << 20  /* Don't propagate to slaves. */
const CLIENT_PREVENT_PROP = CLIENT_PREVENT_AOF_PROP | CLIENT_PREVENT_REPL_PROP
const CLIENT_PENDING_WRITE = 1 << 21 /* Client has output to send but a write
                                        handler is yet not installed. */
const CLIENT_REPLY_OFF = 1 << 22       /* Don't send replies to client. */
const CLIENT_REPLY_SKIP_NEXT = 1 << 23 /* Set CLIENT_REPLY_SKIP for next cmd */
const CLIENT_REPLY_SKIP = 1 << 24      /* Don't send just this reply. */
const CLIENT_LUA_DEBUG = 1 << 25       /* Run EVAL in debug mode. */
const CLIENT_LUA_DEBUG_SYNC = 1 << 26  /* EVAL debugging without fork() */
const CLIENT_MODULE = 1 << 27          /* Non connected client used by some module. */

/* Client request types */
const PROTO_REQ_INLINE = 1
const PROTO_REQ_MULTIBULK = 2

/* Client block type (btype field in client structure)
 * if CLIENT_BLOCKED flag is set. */
const BLOCKED_NONE = 0   /* Not blocked, no CLIENT_BLOCKED flag set. */
const BLOCKED_LIST = 1   /* BLPOP & co. */
const BLOCKED_WAIT = 2   /* WAIT for synchronous replication. */
const BLOCKED_MODULE = 3 /* Blocked by a loadable module. */
const BLOCKED_STREAM = 4 /* XREAD. */
const BLOCKED_ZSET = 5   /* BZPOP et al. */
const BLOCKED_NUM = 6    /* Number of blocked states. */

/* Protocol and I/O related defines */
const PROTO_MAX_QUERYBUF_LEN = 1024 * 1024 * 1024 /* 1GB max query buffer. */
const PROTO_IOBUF_LEN = 1024 * 16                 /* Generic I/O buffer size */
const PROTO_REPLY_CHUNK_BYTES = 16 * 1024         /* 16k output buffer */
const PROTO_INLINE_MAX_SIZE = 1024 * 64           /* Max size of inline reads */
const PROTO_MBULK_BIG_ARG = 1024 * 32
const PROTO_DUMP_LEN = 128
const LONG_STR_SIZE = 21                      /* Bytes needed for long -> str + '\0' */
const REDIS_AUTOSYNC_BYTES = 1024 * 1024 * 32 /* fdatasync every 32MB */

const LIMIT_PENDING_QUERYBUF = 4 * 1024 * 1024 /* 4mb */
const CLIENT_TYPE_NORMAL = 0                   /* Normal req-reply clients + MONITORs */
const CLIENT_TYPE_SLAVE = 1                    /* Slaves. */
const CLIENT_TYPE_PUBSUB = 2                   /* Clients subscribed to PubSub channels. */
const CLIENT_TYPE_MASTER = 3                   /* Master. */
const CLIENT_TYPE_OBUF_COUNT = 4               /* Number of clients to expose to output
                                    buffer configuration. Just the first
                                    three: normal, slave, pubsub. */
/* Networking Constants */

const ANET_OK = 0
const ANET_ERR = -1
const ANET_ERR_LEN = 256

/* Flags used with certain functions. */
const ANET_NONE = 0
const ANET_IP_ONLY = (1 << 0)

const CONFIG_BINDADDR_MAX = 16
const NET_MAX_WRITES_PER_EVENT = 1024 * 64

/* Log levels */
const LL_DEBUG = 0
const LL_INFO = 1
const LL_NOTICE = 2
const LL_WARNING = 3
const LL_RAW = (1 << 10) /* Modifier to log without timestamp */
const CONFIG_DEFAULT_LOGLEVEL = LL_NOTICE

/* Command flags. Please check the command table defined in the redis.c file
 * for more information about the meaning of every flag. */
const CMD_WRITE = 1 << 0              /* "w" flag */
const CMD_READONLY = 1 << 1           /* "r" flag */
const CMD_DENYOOM = 1 << 2            /* "m" flag */
const CMD_MODULE = 1 << 3             /* Command exported by module. */
const CMD_ADMIN = 1 << 4              /* "a" flag */
const CMD_PUBSUB = 1 << 5             /* "p" flag */
const CMD_NOSCRIPT = 1 << 6           /* "s" flag */
const CMD_RANDOM = 1 << 7             /* "R" flag */
const CMD_SORT_FOR_SCRIPT = 1 << 8    /* "S" flag */
const CMD_LOADING = 1 << 9            /* "l" flag */
const CMD_STALE = 1 << 10             /* "t" flag */
const CMD_SKIP_MONITOR = 1 << 11      /* "M" flag */
const CMD_ASKING = 1 << 12            /* "k" flag */
const CMD_FAST = 1 << 13              /* "F" flag */
const CMD_MODULE_GETKEYS = 1 << 14    /* Use the modules getkeys interface. */
const CMD_MODULE_NO_CLUSTER = 1 << 15 /* Deny on Redis Cluster. */

/* Command call flags, see call() function */
const CMD_CALL_NONE = 0
const CMD_CALL_SLOWLOG = 1 << 0
const CMD_CALL_STATS = 1 << 1
const CMD_CALL_PROPAGATE_AOF = 1 << 2
const CMD_CALL_PROPAGATE_REPL = 1 << 3
const CMD_CALL_PROPAGATE = CMD_CALL_PROPAGATE_AOF | CMD_CALL_PROPAGATE_REPL
const CMD_CALL_FULL = CMD_CALL_SLOWLOG | CMD_CALL_STATS | CMD_CALL_PROPAGATE

/* Command propagation flags, see propagate() function */
const PROPAGATE_NONE = 0
const PROPAGATE_AOF = 1
const PROPAGATE_REPL = 2

const DEFAULT_DB_NUM = 16

const CONFIG_DEFAULT_PROTO_MAX_BULK_LEN = 512 * 1024 * 1024
const CONFIG_DEFAULT_MAXMEMORY = 0
const CONFIG_DEFAULT_MAX_CLIENTS = 10000
