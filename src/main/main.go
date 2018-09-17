
package main

import (
	. "kiwi/src/server"
)


func main() {
	s := CreateServer()
	StartServer(s)
	HandleSignal(s)

}

