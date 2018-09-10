package main

import "redigo/src/server"

func main() {
	//ch := make(chan int, 3)
	//go haha1(ch)
	//go haha2(ch)
	//go haha3(ch)
	////ch<-1
	////ch<-1
	////ch<-1
	//close(ch)
	//time.Sleep(time.Second)
	s := server.CreateServer()
	server.StartServer(s)
	server.HandleSignal(s)
}

//func haha1(ch chan int) {
//	var num int
//	select {
//		case num=<-ch:
//			fmt.Println(1, num)
//	}
//}
//func haha2(ch chan int) {
//	var num int
//	select {
//		case num=<-ch:
//			fmt.Println(2, num)
//	}
//}
//func haha3(ch chan int) {
//	var num int
//	select {
//		case num=<-ch:
//			fmt.Println(3, num)
//	}
//}
