package object

import (
	. "redigo/src/structure"
	. "redigo/src/constant"
	"time"
	."redigo/src/server"
	"math"
)

type RInt64 int64
type RString string

type Object struct {
	RType int64
	Encoding int64
	Lru int64
	RefConut int64
}

type IntObject struct {
	Object
	Value *int64
}

type ListObject struct {
	Object
	Value *List
}

type ZSetObject struct {
	Object
	Value *ZSkiplist
}

type HashObject struct {
	Object
	Value *map[string] string
}

type SetObject struct {
	Object
	Value *map[string] string
}

type IObject interface {
	getRType() int64
	getRTypeInString() string
	setRType(rtype int64)
	getEncode() int64
	setEncode(encode int64)
	getEncodeInString() string
	getLRU() int64
	setLRU(lru int64)
	getRefCount() int64
	setRefCount(refCount int64)
	IncrRefCount() int64
	DecrRefCount() int64
	RefreshLRUClock(s *Server)
	//getGetValueFunc() interface{}
	//getSetValueFunc() interface{}
	//isStr() bool
	//getCreateFunc()
}

func (o *Object) getRType() int64 {
	return o.RType
}

func (o *Object) getRTypeInString() string {
	switch o.RType {
	case OBJ_RTYPE_STR:
		return "string"
	case OBJ_RTYPE_INT:
		return "int"
	case OBJ_RTYPE_LIST:
		return "list"
	case OBJ_RTYPE_HASH:
		return "hash"
	case OBJ_RTYPE_SET:
		return "set"
	case OBJ_RTYPE_ZSET:
		return "sorted set"
	default:
		return "unknown"
	}
}

func (o *Object) setRType(rtype int64) {
	o.RType = rtype
}

func (o *Object) getEncode() int64 {
	return o.Encoding
}

func (o *Object) getEncodeInString() string {
	switch o.Encoding {
	case OBJ_ENCODING_STR:
		return "raw"
	case OBJ_ENCODING_INT:
		return "int"
	case OBJ_ENCODING_HT:
		return "hashtable"
	case OBJ_ENCODING_QUICKLIST:
		return "quicklist"
	case OBJ_ENCODING_ZIPLIST:
		return "ziplist"
	case OBJ_ENCODING_INTSET:
		return "intset"
	case OBJ_ENCODING_SKIPLIST:
		return "skiplist"
	default:
		return "unknown"
	}
}

func (o *Object) setEncode(encode int64) {
	o.Encoding = encode
}

func (o *Object) getLRU() int64{
	return o.Lru
}

func (o *Object) setLRU(lru int64) {
	o.Lru = lru
}

func (o *Object) getRefCount() int64 {
	return o.RefConut
}

func (o *Object) setRefCount(refCount int64) {
	o.RefConut = refCount
}

func (o *Object) IncrRefCount() int64 {
	if o.RefConut != math.MaxInt64 {
		o.RefConut--
	}
	return o.RefConut
}

func (o *Object) DecrRefCount() int64 {
	if o.RType == OBJ_RTYPE_STR || o.RType == OBJ_RTYPE_INT {
		if o.RefConut <= 0 {
			panic("DecrRefCount against refcount <= 0")
		}
		if o.RefConut != math.MaxInt64 {
			o.RefConut--
		}
	}
	return o.RefConut
}


func (o *Object) isStr() bool {
	return o.getRType() == OBJ_RTYPE_STR &&o.getEncode() == OBJ_ENCODING_STR
}

func (o *Object) isInt() bool {
	return o.getRType() == OBJ_RTYPE_INT && o.getEncode() == OBJ_ENCODING_INT
}

func (o *Object) RefreshLRUClock(s *Server) {
	o.Lru = LRUClock(s)
}

//
//func (listObj *ListObject) getValue() interface{} {
//	return listObj.Value
//}
//
//func (zsetObj *ZSetObject) getValue() interface{} {
//	return zsetObj.Value
//}
//
//func (hashObj *HashObject) getValue() interface{} {
//	return hashObj.Value
//}
//
//func (setObj *SetObject) getValue() interface{} {
//	return setObj.Value
//}
//
//func (strObj *StrObject) setValue(str string) bool {
//	strObj.Value
//}
//
//func (listObj *ListObject) setValue() bool {
//	return listObj.Value
//}
//
//func (zsetObj *ZSetObject) setValue() bool {
//	return zsetObj.Value
//}
//
//func (hashObj *HashObject) setValue() bool {
//	return hashObj.Value
//}
//
//func (setObj *SetObject) setValue() bool {
//	return setObj.Value
//}

/* functions for Objects */
func createObject(s *Server, rtype int64, encoding int64,) Object{
	obj := Object {
		RType:rtype,
		Encoding:encoding,
		Lru: LRUClock(s),
		RefConut:1,
	}
	return obj
}

func LRUClock(s *Server) int64 {
	if 1000/s.Hz <= LRU_CLOCK_RESOLUTION {
		// server.Hz >= 1, serverCron will update LRU, save resources
		return s.LruClock
	} else {
		return SimpleGetLRUClock()
	}
}

func SimpleGetLRUClock() int64 {
	mstime := time.Now().UnixNano()/1000
	return mstime / LRU_CLOCK_RESOLUTION & LRU_CLOCK_MAX
}



