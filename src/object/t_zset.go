package object

import (
	."redigo/src/server"
	."redigo/src/constant"
)

type ZSet struct {
	RedisObject
}


/*-----------------------------------------------------------------------------
 * Common sorted set API
 *----------------------------------------------------------------------------*/

func (zset *ZSet) ZSetLength() int {
	length := 0
	if zset.Encoding == OBJ_ENCODING_SKIPLIST {
		length =  zset.Ptr.(*ZSkiplist)
	}
}
