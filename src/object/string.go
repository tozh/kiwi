package object

import (
	. "redigo/src/constant"
	."redigo/src/server"
	"strconv"
	"math"
	"errors"
	"bytes"
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

func CatString(a string, b string) string{
	if b == "" {
		return a
	}
	buf := bytes.Buffer{}
	buf.WriteString(a)
	buf.WriteString(b)
	return buf.String()

}


func IsStrObjectInt64(o *StrObject) bool {
	return o != nil && o.RType == OBJ_RTYPE_STR && o.Encoding == OBJ_ENCODING_INT
}

func IsStrObjectString(o *StrObject) bool {
	return o != nil && o.RType == OBJ_RTYPE_STR && o.Encoding == OBJ_ENCODING_STR
}

func GetStrObjectValueInt64(o *StrObject) (int64, error) {
	if IsStrObjectInt64(o) {
		return *o.Value.(*int64), nil
	}
	return 0, errors.New("not int64 StrObject")
}

func GetStrObjectValueString(o *StrObject) (string, error) {
	if IsStrObjectString(o) {
		return *o.Value.(*string), nil
	}
	if IsStrObjectInt64(o) {
		return strconv.FormatInt(*o.Value.(*int64), 10), nil
	}
	return "", errors.New("not StrObject")
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

func ReplaceStrObjectByInt64(s *Server, o *StrObject, oldValue *int64, newValue *int64) *StrObject {
	if !IsSharedInt64(*oldValue) && !IsSharedInt64(*newValue) {
		o.Value = newValue
		o.RefreshLRUClock(s)
		return o
	} else {
		o.DecrRefCount()
		return CreateStrObjectByInt64(s, *newValue)
	}
}

func AppendStrObject(s *Server, o *StrObject, b string) *StrObject {
	if b == "" {
		return o
	}
	if IsStrObjectString(o) {
		str := CatString(*o.Value.(*string), b)
		o.Value = &str
	}
	if IsStrObjectInt64(o) {
		str := strconv.FormatInt(*o.Value.(*int64), 10)
		str = CatString(str, b)
		o.Value = &str
		o.setEncode(OBJ_ENCODING_STR)
	}

	return StrObjectEncode(s, o)
}


func StrObjectEncode(s *Server, o *StrObject) *StrObject {
	if !IsStrObjectString(o) || o.RefConut > 1 {
		return o
	}

	i, err := strconv.ParseInt(*o.Value.(*string), 10, 64)
	if err == nil {
		if IsSharedInt64(i) {
			o.DecrRefCount()
			s.Shared.Integers[i].IncrRefCount()
			return s.Shared.Integers[i]
		} else {
			o.Value = &i
			o.setEncode(OBJ_ENCODING_INT)
		}
	}
	return o
}

/* Get a decoded version of an encoded object (returned as a new object).
 * If the object is already raw-encoded just increment the ref count. */
func StrObjectDecode(s *Server, o *StrObject) *StrObject {
	if IsStrObjectInt64(o) {
		str := strconv.FormatInt(*o.Value.(*int64), 10)
		return CreateStrObjectByStr(s, str)
	}
	return o
}

