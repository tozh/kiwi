package server

import (
	"time"
)

type Object struct {
	OType    byte
	Encoding byte
	Lru      time.Time
	//RefConut int
}

type IntObject struct {
	Object
	Value *int
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
	getOType() byte
	getOTypeInString() string
	setOType(otype byte)
	getEncode() byte
	setEncode(encode byte)
	getEncodeInString() string
	getLRU() time.Time
	setLRU(lru time.Time)
	//getRefCount() int
	//setRefCount(refCount int)
	//IncrRefCount() int
	//DecrRefCount() int
	RefreshLRUClock()
}

func (o *Object) getOType() byte {
	return o.OType
}

func (o *Object) getOTypeInString() string {
	switch o.OType {
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

func (o *Object) setOType(otype byte) {
	o.OType = otype
}

func (o *Object) getEncode() byte {
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

func (o *Object) setEncode(encode byte) {
	o.Encoding = encode
}

func (o *Object) getLRU() time.Time {
	return o.Lru
}

func (o *Object) setLRU(lru time.Time) {
	o.Lru = lru
}

//func (o *Object) getRefCount() int {
//	return o.RefConut
//}
//
//func (o *Object) setRefCount(refCount int) {
//	o.RefConut = refCount
//}
//
//func (o *Object) IncrRefCount() int {
//	if o.RefConut != math.MaxInt64 {
//		o.RefConut--
//	}
//	return o.RefConut
//}
//
//func (o *Object) DecrRefCount() int {
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

func (o *Object) RefreshLRUClock() {
	o.Lru = LruClock(kiwiS)
}

/* functions for Objects */
func CreateObject(otype byte, encoding byte) Object {
	obj := Object{
		OType:    otype,
		Encoding: encoding,
		Lru:      LruClock(kiwiS),
		//RefConut: 1,
	}
	return obj
}

func CheckOType(o Objector, otype byte) bool {
	return o != nil && o.getOType() == otype
}
