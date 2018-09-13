package server

import (
	"fmt"
)

type SharedObjects struct {
	Crlf           string // "\r\n"
	NullBulk       string // "$-1\r\n"
	EmptyBulk      string // "$0\r\n"
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
	MultiBulkHDR   [SHARED_BULKHDR_LEN]string // "*<value>\r\n"
	BulkHDR        [SHARED_BULKHDR_LEN]string // "$<value>\r\n"
}

func CreateShared(s *Server) *SharedObjects {
	s.ServerLogDebugF("-->%v\n", "CreateShared")

	so := SharedObjects{
		Crlf:           "\r\n",
		NullBulk:       "$-1\r\n",
		EmptyBulk:      "$0\r\n",
		NullMultiBulk:  "*-1\r\n",
		EmptyMultiBulk: "*0\r\n",
		Zero:           ":0\r\n",
		One:            ":1\r\n",
		NegOne:         ":-1\r\n",
		Ok:             "+OK\r\n",
		Err:            "-ERR\r\n",
		NoAuthErr:      "-NOAUTH Authentication required.\r\n",
		OOMErr:         "-OOM command not allowed when used memory > 'maxmemory'.\r\n",
		LoadingErr:     "-LOADING Redis is loading the dataset in memory\r\n",
		SyntaxErr:      "-ERR syntax error\r\n",
		WrongTypeErr:   "-WRONGTYPE Operation against a key holding the wrong kind of value\r\n",
		Integers:       [SHARED_INTEGERS]*StrObject{},
		MultiBulkHDR:   [SHARED_BULKHDR_LEN]string{}, // "*<value>\r\n"
		BulkHDR:        [SHARED_BULKHDR_LEN]string{}, // "$<value>\r\n"
	}
	for i:=0; i<SHARED_INTEGERS; i++ {
		so.Integers[i] = &StrObject{
			CreateObject(s, OBJ_RTYPE_STR, OBJ_ENCODING_INT),
			&i,
		}
	}
	for i:=0; i<SHARED_BULKHDR_LEN; i++ {
		so.MultiBulkHDR[i] = fmt.Sprintf("*%d\r\n", i)
	}
	for i:=0; i<SHARED_BULKHDR_LEN; i++ {
		so.BulkHDR[i] = fmt.Sprintf("$%d\r\n", i)
	}
	s.Shared = &so
	return s.Shared
}
