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
	IncrRefCount() int
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


