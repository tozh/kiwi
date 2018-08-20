package constant

/* constants for linkedlist */
const ITERATION_DIRECTION_INORDER = 1
const ITERATION_DIRECTION_REVERSE_ORDER = -1

/* constants for zskiplist */
const ZSKIPLIST_MAXLEVEL = 64
const ZSKIPLIST_P = 0.25
const ZSKIPLIST_RANDOM_MAXLEVEL = 0xFFFF * ZSKIPLIST_P

/* constants for Object */
const OBJ_ENCODING_RAW = 0
const OBJ_ENCODING_INT = 1
const OBJ_ENCODING_HT = 2
const OBJ_ENCODING_ZIPMAP = 3
const OBJ_ENCODING_LINKEDLIST = 4
const OBJ_ENCODING_ZIPLIST = 5
const OBJ_ENCODING_INTSET = 6
const OBJ_ENCODING_SKIPLIST = 7
const OBJ_ENCODING_EMBSTR = 8
const OBJ_ENCODING_QUICKLIST = 9
const OBJ_ENCODING_STREAM = 10

const OBJ_RTYPE_STR = 0
const OBJ_RTYPE_INT = 1
const OBJ_RTYPE_LIST = 2
const OBJ_RTYPE_ZSET = 3
const OBJ_RTYPE_HASH = 4
const OBJ_RTYPE_SET = 5


const DICT_ON = 0
const DICT_ERR = 1

const LRU_BITS = 24
const LRU_CLOCK_MAX = (1 << LRU_BITS) - 1  /* Max value of obj->lru */
const LRU_CLOCK_RESOLUTION = 1000  /* LRU clock resolution in ms */
