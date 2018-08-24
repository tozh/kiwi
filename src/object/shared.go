package object

import ."redigo/src/constant"

type SharedObjects struct {
	Integers [SHARED_INTEGERS]*StrObject
	NullBulk *StrObject
	WrongType *StrObject
	CommandZero *StrObject
	CommandOne *StrObject
	CommandOk *StrObject
	CommandErr *StrObject
}
