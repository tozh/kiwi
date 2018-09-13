package main

import ."redigo/src/test_server"

func main() {
	s := CreateServer()
	StartServer(s)
	HandleSignal(s)
}

