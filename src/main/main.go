package main

import (
	"fmt"
	"time"
)

func main() {
	ch := make(chan int)
	go haha1(ch)
	go haha2(ch)
	go haha3(ch)
	time.Sleep(time.Second)
	ch<-1
	time.Sleep(time.Second)
	ch<-1
	time.Sleep(time.Second)
	time.Sleep(time.Second)
	time.Sleep(time.Second)
}

func haha1(ch chan int) {
	var num int
	select {
		case num=<-ch:
			fmt.Println(1, num)
	}
}
func haha2(ch chan int) {
	var num int
	select {
		case num=<-ch:
			fmt.Println(2, num)
	}
}
func haha3(ch chan int) {
	var num int
	select {
		case num=<-ch:
			fmt.Println(3, num)
	}
}
