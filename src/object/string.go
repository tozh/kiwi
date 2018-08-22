package object

import (
	. "redigo/src/constant"
	."redigo/src/server"
	"strconv"
	"math"
)


type StrObject struct {
	Object
	Value interface{}
}

func IsSharedInt64(i int64) bool {
	return 0 <= i && i < SHARED_INTEGERS
}

func IsOverflowInt64(oldValue int64, incr int64) bool{
	return (incr < 0 && oldValue < 0 && incr < math.MinInt64 - oldValue) ||
		(incr > 0 && oldValue > 0 && incr > math.MaxInt64 - oldValue)
}

func CreateStrObjectByStr(s *Server, str string) *StrObject {
	obj := createObject(s, OBJ_RTYPE_STR, OBJ_ENCODING_STR)
	o := StrObject {
		Object:obj,
		Value:&str,
	}
	return StrObjectEncode(s, &o)
}

func CreateStrObjectByInt64(s *Server, i int64) *StrObject {
	if IsSharedInt64(i) {
		o := s.Shared.Integers[i]
		o.IncrRefCount()
		return o
	}
	obj := createObject(s, OBJ_RTYPE_STR, OBJ_ENCODING_INT)
	o := StrObject {
		Object:obj,
		Value:&i,
	}
	return &o
}

func ReplaceStrObjectByInt64(s *Server, o *StrObject, oldValue int64, newValue int64) *StrObject {
	if !IsSharedInt64(oldValue) && !IsSharedInt64(newValue) {
		o.Value = &newValue
		o.RefreshLRUClock(s)
		return o
	} else {
		o.DecrRefCount()
		return CreateStrObjectByInt64(s, newValue)
	}

}

func StrObjectEncode(s *Server, o *StrObject) *StrObject {
	if o.Encoding != OBJ_ENCODING_STR || o.RefConut > 1 {
		return o
	}

	str := *o.Value.(*string)
	i, err := strconv.ParseInt(str, 10, 64)
	if err == nil {
		if IsSharedInt64(i) {
			o.DecrRefCount()
			s.Shared.Integers[i].IncrRefCount()
			return s.Shared.Integers[i]
		} else {
			o.Value = &i
			o.Encoding = OBJ_ENCODING_INT
		}
	}
	return o
}

/* Get a decoded version of an encoded object (returned as a new object).
 * If the object is already raw-encoded just increment the ref count. */
func StrObjectDecode(s *Server, o *StrObject) *StrObject {
	if o.RType == OBJ_RTYPE_STR && o.Encoding == OBJ_ENCODING_INT {
		str := strconv.FormatInt(*o.Value.(*int64), 10)
		return CreateStrObjectByStr(s, str)
	}
	o.IncrRefCount()
	return o
}

