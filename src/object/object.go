package object

import (
	. "redigo/src/structure"
)

type Object struct {
	RType int
	Encoding int
	Lru int
	RefConut int
}

type StrObject struct {
	Object
	Value string
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
	getRType() int
	setRType(rtype int)
	getEncode() int
	setEncode(encode int)
	getLRU() int
	setLRU(lru int)
	getRefCount() int
	setRefCount(refCount int)
	IncrRefCount(count int) int
}

type IObjectValue interface {
	getGetValueFunc() interface{}
	getSetValueFunc() interface{}
}

func (obj *Object) getEncode() int{
	return obj.Encoding
}

func (obj *Object) setEncode(encode int) {
	obj.Encoding = encode
}

func (obj *Object) getRType() int {
	return obj.RType
}

func (obj *Object) setRType(rtype int) {
	obj.RType = rtype
}

func (obj *Object) getLRU() int{
	return obj.Lru
}

func (obj *Object) setLRU(lru int) {
	obj.Lru = lru
}

func (obj *Object) getRefCount() int {
	return obj.RefConut
}

func (obj *Object) setRefCount(refCount int) {
	obj.RefConut = refCount
}

func (obj *Object) IncrRefCount(count int) int {
	obj.RefConut += count
	return obj.RefConut
}

func (strObj *StrObject) getValue() string {
	return strObj.Value
}

func (strObj *StrObject) setValue(str string) bool {
	strObj.Value = str
	return true
}

func (strObj *StrObject) getGetValueFunc() interface{} {
	return strObj.getValue
}

func (strObj *StrObject) getSetValueFunc() interface{} {
	return strObj.setValue
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

