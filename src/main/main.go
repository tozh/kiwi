package main

import (
	. "kiwi/src/server"
	"fmt"
)

func main() {
	//testBuffer()
	s := CreateServer()
	StartServer(s)
	HandleSignal(s)
}

func testBuffer() {
	sbuf := Buffer{}
	sbuf.WriteString("hahahahah")
	sbuf.WriteString("heheheheh")
	fmt.Println(sbuf.String())
}

