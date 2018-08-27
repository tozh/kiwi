package object

import ."redigo/src/constant"

type SharedObjects struct {
	Integers [SHARED_INTEGERS]*StrObject
	Crlf string  // "\r\n"
	NullBulk string  // "$-1\r\n"
	NullMultiBulk string  // "*-1\r\n"
	EmptyMultiBulk string // "*0\r\n"
	Zero string  // ":0\r\n"
	One string  // ":1\r\n"
	NegOne string  // ":-1\r\n"
	Ok string  // "+OK\r\n"
	Err string  // "-ERR\r\n"
	MultiBulkHDR [SHARED_BULKHDR_LEN]string  // "$<value>\r\n"
	BulkHDR [SHARED_BULKHDR_LEN]string // "$<value>\r\n"
	WrongType *StrObject

}
