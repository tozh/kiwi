package object

import (
	. "redigo/src/structure"
	. "redigo/src/constant"
	"time"
	."redigo/src/server"
	"strconv"
)

type Object struct {
	RType int64
	Encoding int64
	Lru int64
	RefConut int64
}

type StrObject struct {
	Object
	Value *string
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
	IncrRefCount(count int64) int64
	getGetValueFunc() interface{}
	getSetValueFunc() interface{}
	//getCreateFunc()
}

func (obj *Object) getRType() int64 {
	return obj.RType
}

func (obj *Object) getRTypeInString() string {
	switch obj.RType {
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

func (obj *Object) setRType(rtype int64) {
	obj.RType = rtype
}

func (obj *Object) getEncode() int64 {
	return obj.Encoding
}

func (obj *Object) getEncodeInString() string {
	switch obj.Encoding {
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
	case OBJ_ENCODING_EMBSTR:
		return "embstr"
	default:
		return "unknown"
	}
}

func (obj *Object) setEncode(encode int64) {
	obj.Encoding = encode
}

func (obj *Object) getLRU() int64{
	return obj.Lru
}

func (obj *Object) setLRU(lru int64) {
	obj.Lru = lru
}

func (obj *Object) getRefCount() int64 {
	return obj.RefConut
}

func (obj *Object) setRefCount(refCount int64) {
	obj.RefConut = refCount
}

func (obj *Object) IncrRefCount(count int64) int64 {
	obj.RefConut += count
	return obj.RefConut
}


func (strObj *StrObject) getValue() string {
	return *strObj.Value
}

func (strObj *StrObject) setValue(str string) bool {
	strObj.Value = &str
	strObj.setRType(OBJ_RTYPE_STR)
	return true
}

func (strObj *StrObject) getGetValueFunc() interface{} {
	return strObj.getValue
}

func (strObj *StrObject) getSetValueFunc() interface{} {
	return strObj.setValue
}


func (intObj *IntObject)  getValue() int64 {
	return *intObj.Value
}

func (intObj *IntObject) setValue(num int64) bool {
	intObj.Value = &num
	intObj.RType = OBJ_RTYPE_INT
	return true
}

func (intObj *IntObject) getGetValueFunc() interface{} {
	return intObj.getValue
}

func (intObj *IntObject) getSetValueFunc() interface{} {
	return intObj.setValue
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
func createObject(rtype int64, encoding int64,server *Server) Object{
	obj := Object {
		RType:rtype,
		Encoding:encoding,
		Lru: LRUClock(server),
		RefConut:1,
	}
	return obj
}

func createStrObject(str string, server *Server) IObject {
	obj := createObject(OBJ_RTYPE_STR, OBJ_ENCODING_STR, server)
	strObj := StrObject {
		Object:obj,
		Value:&str,
	}
	return &strObj
}

func createIntObject(num int64, server *Server) IObject {
	obj := createObject(OBJ_RTYPE_INT, OBJ_ENCODING_INT, server)
	strObj := IntObject {
		Object:obj,
		Value:&num,
	}
	return &strObj
}

func LRUClock(server *Server) int64 {
	if 1000/server.Hz <= LRU_CLOCK_RESOLUTION {
		// server.Hz >= 1, serverCron will update LRU, save resources
		return server.LruClock
	} else {
		return SimpleGetLRUClock()
	}
}

func SimpleGetLRUClock() int64 {
	mstime := time.Now().UnixNano()/1000
	return mstime / LRU_CLOCK_RESOLUTION & LRU_CLOCK_MAX
}

func TryObjectEncoding(o IObject) IObject {
	if !(o.getEncode() == OBJ_ENCODING_STR||o.getEncode() == OBJ_ENCODING_EMBSTR) {
		return o
	}
	if o.getRefCount() > 1 {
		return o
	}
	//length := len(*o.(*StrObject).Value)
	value, err := strconv.ParseInt(*o.(*StrObject).Value, 10, 64)
	if err == nil {
		if o.getEncode() == OBJ_ENCODING_STR {
			o.Value
		}
	}
}

