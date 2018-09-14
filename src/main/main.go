package main

import (
	. "KiwiDB/src/server"
)

func main() {
	s := CreateServer()
	StartServer(s)
	HandleSignal(s)

	//buf := bytes.Buffer{}
	////buf.WriteString("*3\r\n$3\r\nset\r\n$3\r\nfoo\r\n$3\r\nbar\r\n")
	//
	//bs := []byte("12345678")
	//bs = append(bs, 0)
	//fmt.Println(string(bs))
	//
	//reader := bytes.NewReader(bs)
	//reader.

	//if b!='*' {
	//	fmt.Println("error protocol format")
	//}
	//bs, _ := buf.ReadBytes('\r')
	//
	//multiBulkLen, _ := strconv.Atoi(string(bs[:len(bs)-1]))
	//fmt.Println(multiBulkLen)
	//b, _ = buf.ReadByte()
	//fmt.Println(b == '\n')
	//for multiBulkLen > 0 {
	//	multiBulkLen--
	//	//bs, _ := buf.ReadBytes('\r')
	//	//bulkLen, _ := strconv.Atoi(string(bs))
	//	//buf.ReadByte()
	//
	//}
	//a := make([]byte, 0)
	//a = (a, "aaaa"...)
	//fmt.Println(a)
}

