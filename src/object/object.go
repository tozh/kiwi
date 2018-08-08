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
	Value List
}

type ZSetObject struct {

}

type HashObject struct {
	Object
	Value map [string] string
}

type SetObject struct {
	Object
	Value map [string] string
}

type object interface {
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



