package object

import (
	. "redigo/src/structure"
	. "redigo/src/constant"
	."redigo/src/server"
	"strconv"
)

type StrObject struct {
	Object
	Value *string
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

func TryObjectEncode(s *Server, o IObject) IObject {
	if o.getEncode() != OBJ_ENCODING_STR {
		return o
	}
	if o.getRefCount() > 1 {
		return o
	}
	value, err := strconv.ParseInt(*o.(*StrObject).Value, 10, 64)
	if err != nil {
		if value >= 0 && value < SHARED_INTEGERS {
			o.decrRefCount()
			s.Shared.Integers[value].incrRefCount()
			return s.Shared.Integers[value]
		} else {
			return StrObjectToIntObject(o.(*StrObject), value)
		}
	}
	return o
}


func StrObjectToIntObject(o *StrObject, value int64) *IntObject{
	return &IntObject{
		Object {
			OBJ_RTYPE_INT,
			OBJ_ENCODING_INT,
			o.Lru,
			o.RefConut,
		},
		&value,
	}
}

/* Get a decoded version of an encoded object (returned as a new object).
 * If the object is already raw-encoded just increment the ref count. */
func ObjectDecode(s *Server, o IObject) IObject {
	if o.isStr() {
		o.incrRefCount()
		return o
	}
	if
}