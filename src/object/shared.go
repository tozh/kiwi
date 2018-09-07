package object

import . "redigo/src/constant"

type SharedObjects struct {
	Crlf           string // "\r\n"
	NullBulk       string // "$-1\r\n"
	NullMultiBulk  string // "*-1\r\n"
	EmptyMultiBulk string // "*0\r\n"
	Zero           string // ":0\r\n"
	One            string // ":1\r\n"
	NegOne         string // ":-1\r\n"
	Ok             string // "+OK\r\n"
	Err            string // "-ERR\r\n"
	NoAuthErr      string // "-NOAUTH Authentication required.\r\n"
	OOMErr         string // "-OOM command not allowed when used memory > 'maxmemory'.\r\n"
	LoadingErr     string // "-LOADING Redis is loading the dataset in memory\r\n"
	SyntaxErr      string // "-ERR syntax error\r\n"
	WrongTypeErr   string // "-WRONGTYPE Operation against a key holding the wrong kind of value\r\n"
	Integers       [SHARED_INTEGERS]*StrObject
	MultiBulkHDR   [SHARED_BULKHDR_LEN]string // "$<value>\r\n"
	BulkHDR        [SHARED_BULKHDR_LEN]string // "$<value>\r\n"
	WrongType      *StrObject
}
