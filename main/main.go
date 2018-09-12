package main

import ."redigo/src/test_server"

func main() {

	//time.Sleep(time.Second)
	//s := server.CreateServer()
	//server.StartServer(s)
	//server.HandleSignal(s)

	s := CreateServer()
	StartServer(s)
	HandleSignal(s)

	//Connect()
}
