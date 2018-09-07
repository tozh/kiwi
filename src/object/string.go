package object

import (
	. "redigo/src/constant"
	. "redigo/src/server"
	"strconv"
	"math"
	"errors"
	"bytes"
)

type StrObject struct {
	Object
	Value interface{}
}

func IsSharedInt(i int64) bool {
	return 0 <= i && i < SHARED_INTEGERS
}

func IsOverflowInt(oldValue int64, incr int64) bool {
	return (incr < 0 && oldValue < 0 && incr < math.MinInt64-oldValue) ||
		(incr > 0 && oldValue > 0 && incr > math.MaxInt64-oldValue)
}

func IsStrObjectInt(o *StrObject) bool {
	return o != nil && o.RType == OBJ_RTYPE_STR && o.Encoding == OBJ_ENCODING_INT
}

func IsStrObjectString(o *StrObject) bool {
	return o != nil && o.RType == OBJ_RTYPE_STR && o.Encoding == OBJ_ENCODING_STR
}

func GetStrObjectValueInt(o *StrObject) (int64, error) {
	if IsStrObjectInt(o) {
		return *o.Value.(*int64), nil
	}
	return 0, errors.New("not int64 StrObject")
}

func GetStrObjectValueString(o *StrObject) (string, error) {
	if IsStrObjectString(o) {
		return *o.Value.(*string), nil
	}
	if IsStrObjectInt(o) {
		return strconv.FormatInt(*o.Value.(*int64), 10), nil
	}
	return "", errors.New("not StrObject")
}

func CreateStrObjectByStr(s *Server, str string) *StrObject {
	obj := createObject(s, OBJ_RTYPE_STR, OBJ_ENCODING_STR)
	o := StrObject{
		Object: obj,
		Value:  &str,
	}
	return StrObjectEncode(s, &o)
}

func CreateStrObjectByInt(s *Server, i int64) *StrObject {
	if IsSharedInt(i) {
		o := s.Shared.Integers[i]
		//o.IncrRefCount()
		return o
	}
	obj := createObject(s, OBJ_RTYPE_STR, OBJ_ENCODING_INT)
	o := StrObject{
		Object: obj,
		Value:  &i,
	}
	return &o
}

func ReplaceStrObjectByInt(s *Server, o *StrObject, oldValue *int64, newValue *int64) *StrObject {
	if !IsSharedInt(*oldValue) && !IsSharedInt(*newValue) {
		o.Value = newValue
		o.RefreshLRUClock(s)
		return o
	} else {
		//o.DecrRefCount()
		return CreateStrObjectByInt(s, *newValue)
	}
}

func AppendStrObject(s *Server, o *StrObject, b string) (*StrObject, int64) {
	var length int64
	if b == "" {
		length = StrObjectLength(o)
		return o, length
	}
	if IsStrObjectString(o) {
		str := CatString(*o.Value.(*string), b)
		o.Value = &str
		length = int64(len(str))
	}
	if IsStrObjectInt(o) {
		str := strconv.FormatInt(*o.Value.(*int64), 10)
		str = CatString(str, b)
		o.Value = &str
		o.setEncode(OBJ_ENCODING_STR)
		length = int64(len(str))
	}
	return StrObjectEncode(s, o), length
}

func StrObjectEncode(s *Server, o *StrObject) *StrObject {
	if !IsStrObjectString(o) {
		return o
	}

	i, err := strconv.ParseInt(*o.Value.(*string), 10, 64)
	if err == nil {
		if IsSharedInt(i) {
			//o.DecrRefCount()
			//s.Shared.Integers[i].IncrRefCount()
			return s.Shared.Integers[i]
		} else {
			o.Value = &i
			o.setEncode(OBJ_ENCODING_INT)
		}
	}
	return o
}

func StrObjectLength(o *StrObject) int64 {
	if o.RType != OBJ_RTYPE_STR {
		return 0
	}
	if o.Encoding == OBJ_ENCODING_STR {
		return int64(len(*o.Value.(*string)))
	} else if o.Encoding == OBJ_ENCODING_INT {
		str := strconv.FormatInt(*o.Value.(*int64), 10)
		return int64(len(str))
	}
	return 0
}

/* Get a decoded version of an encoded object (returned as a new object).
 * If the object is already raw-encoded just increment the ref count. */
func StrObjectDecode(s *Server, o *StrObject) *StrObject {
	if IsStrObjectInt(o) {
		str := strconv.FormatInt(*o.Value.(*int64), 10)
		return CreateStrObjectByStr(s, str)
	}
	return o
}

// Utilities for string
func IsSpace(b byte) bool {
	return b == ' ' || b == '\r' || b == '\n'
}

func CatString(a string, b string) string {
	if b == "" {
		return a
	}
	buf := bytes.Buffer{}
	buf.WriteString(a)
	buf.WriteString(b)
	return buf.String()
}

/* Split a line into arguments, where every argument can be in the
 * following programming-language REPL-alike form:
 *
 * foo bar "newline are supported\n" and "\xff\x00otherstuff"
 *
 * The number of arguments is stored into *argc, and an array
 * of sds is returned.
 *
 * The caller should free the resulting array of sds strings with
 * sdsfreesplitres().
 *
 * Note that sdscatrepr() is able to convert back a string into
 * a quoted string in the same format sdssplitargs() is able to parse.
 *
 * The function returns the allocated tokens on success, even when the
 * input string is empty, or NULL if the input contains unbalanced
 * quotes or closed quotes followed by non space characters
 * as in: "foo"bar or "foo'
 */
func SplitArgs(args []byte) []string {
	var vector []string
	for i := 0; i < len(args); i++ {
		// skip blanks
		for i < len(args) && IsSpace(args[i]) {
			i++
		}
		if i < len(args) {
			inQoutes := false
			inSingleQoutes := false
			done := false
			buf := bytes.Buffer{}
			for !done {
				if inQoutes {
					if args[i] == '\\' && args[i+1] == 'x' && IsHexDigit(args[i+2]) && IsHexDigit(args[i+3]) {
						b := HexDigitToInt(args[i+2])*16 + HexDigitToInt(args[i+3])
						buf.WriteByte(b)
						i += 3
					} else if args[i] == '\\' && i+1 < len(args) {
						i++
						switch args[i] {
						case 'n':
							buf.WriteByte('\n')
						case 'r':
							buf.WriteByte('\r')
						case 't':
							buf.WriteByte('\t')
						case 'b':
							buf.WriteByte('\b')
						case 'a':
							buf.WriteByte('\a')
						default:
							buf.WriteByte(args[i])
						}
					} else if args[i] == '"' {
						// closing quote must be followed by a space or nothing at all.
						if i+1 < len(args) && !IsSpace(args[i+1]) {
							return nil
						}
						done = true
					} else if i >= len(args) {
						/* unterminated quotes */
						return nil
					} else {
						buf.WriteByte(args[i])
					}
				} else if inSingleQoutes {
					if args[i] == '\\' && args[i+1] == '\'' {
						buf.WriteByte('\'')
						i++
					} else if args[i] == '\'' {
						// closing quote must be followed by a space or nothing at all.
						if i+1 < len(args) && !IsSpace(args[i+1]) {
							return nil
						}
						done = true
					} else if i >= len(args) {
						return nil
					} else {
						buf.WriteByte(args[i])
					}
				} else {
					switch args[i] {
					case ' ':
						done = true
					case '\n':
						done = true
					case '\r':
						done = true
					case '\t':
						done = true
					case 0:
						done = true
					case '"':
						inQoutes = true
					case '\'':
						inSingleQoutes = true
					default:
						buf.WriteByte(args[i])
					}
				}
				if i < len(args) {
					i++
				}
			}
			vector = append(vector, buf.String())
		}
	}
	return vector
}

func IsHexDigit(b byte) bool {
	return ('0' <= b && b <= '9') || ('a' <= b && b <= 'f') || ('A' <= b && b <= 'F')
}

func HexDigitToInt(b byte) byte {
	switch b {
	case '0':
		return 0
	case '1':
		return 1
	case '2':
		return 2
	case '3':
		return 3
	case '4':
		return 4
	case '5':
		return 5
	case '6':
		return 6
	case '7':
		return 7
	case '8':
		return 8
	case '9':
		return 9
	case 'A':
		return 10
	case 'a':
		return 10
	case 'B':
		return 11
	case 'b':
		return 11
	case 'C':
		return 12
	case 'c':
		return 12
	case 'D':
		return 13
	case 'd':
		return 13
	case 'E':
		return 14
	case 'e':
		return 14
	case 'F':
		return 15
	case 'f':
		return 15
	default:
		return 0
	}
}
