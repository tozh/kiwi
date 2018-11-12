package main

import "kiwi/src/server"

func main() {
	server.InitServer()
	server.StartServer()
	server.WaitEventServerClosed()
	server.CloseServer()
}