package server

import (
	. "redigo/src/constant"
	. "redigo/src/structure"
	"time"
)

type Object struct {
	RType    int64
	Encoding int64
	Lru      time.Time
	//RefConut int64
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
	Value *map[string]string
}

type SetObject struct {
	Object
	Value *map[string]string
}

type Objector interface {
	getRType() int64
	getRTypeInString() string
	setRType(rtype int64)
	getEncode() int64
	setEncode(encode int64)
	getEncodeInString() string
	getLRU() time.Time
	setLRU(lru time.Time)
	//getRefCount() int64
	//setRefCount(refCount int64)
	//IncrRefCount() int64
	//DecrRefCount() int64
	RefreshLRUClock(s *Server)
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

func (o *Object) getLRU() time.Time {
	return o.Lru
}

func (o *Object) setLRU(lru time.Time) {
	o.Lru = lru
}

//func (o *Object) getRefCount() int64 {
//	return o.RefConut
//}
//
//func (o *Object) setRefCount(refCount int64) {
//	o.RefConut = refCount
//}
//
//func (o *Object) IncrRefCount() int64 {
//	if o.RefConut != math.MaxInt64 {
//		o.RefConut--
//	}
//	return o.RefConut
//}
//
//func (o *Object) DecrRefCount() int64 {
//	if o.RType == OBJ_RTYPE_STR || o.RType == OBJ_RTYPE_INT {
//		if o.RefConut <= 0 {
//			panic("DecrRefCount against refcount <= 0")
//		}
//		if o.RefConut != math.MaxInt64 {
//			o.RefConut--
//		}
//	}
//	return o.RefConut
//}

func (o *Object) RefreshLRUClock(s *Server) {
	o.Lru = LruClock(s)
}

/* functions for Objects */
func CreateObject(s *Server, rtype int64, encoding int64) Object {
	obj := Object{
		RType:    rtype,
		Encoding: encoding,
		Lru:      LruClock(s),
		//RefConut: 1,
	}
	return obj
}


func CheckRType(o Objector, rtype int64) bool {
	return o != nil && o.(*Object).RType == rtype
}
