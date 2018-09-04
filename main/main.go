package main

import (
	"fmt"
	"bytes"
)

type data struct {
	//count int64
	//sync.Mutex
	Buf    []byte
	BufPos int64
}

func main() {
	//d := data {
	//	make([]byte, 20),
	//	0,
	//}
	//fmt.Println(&d.Buf[0])
	//fmt.Println(cap(d.Buf))
	//copy(d.Buf[d.BufPos:], "hello")
	//d.BufPos += int64(len("hello"))
	//fmt.Println(cap(d.Buf))
	//fmt.Println(&d.Buf[0])
	//fmt.Println(string(d.Buf))
	//
	//fmt.Println(&d.Buf[0])
	//fmt.Println(cap(d.Buf))
	//copy(d.Buf[d.BufPos:], " \r\n")
	//d.BufPos += int64(len(" \r\n"))
	//fmt.Println(cap(d.Buf))
	//fmt.Println(&d.Buf[0])
	//fmt.Println(string(d.Buf))

	//chans := make([]chan int64, 10)
	//t := time.Now()
	//d := data{
	//	0,
	//	sync.Mutex{},
	//}
	//for i:=0;i<10;i++ {
	//	chans[i] = make(chan int64)
	//	go d.getCount(chans[i], i)
	//}
	//fmt.Println("i: 0----->", <-chans[0])
	//fmt.Println("i: 1----->", <-chans[1])
	//fmt.Println("i: 2----->", <-chans[2])
	//fmt.Println("i: 3----->", <-chans[3])
	//fmt.Println("i: 4----->", <-chans[4])
	//fmt.Println("i: 5----->", <-chans[5])
	//fmt.Println("i: 6----->", <-chans[6])
	//fmt.Println("i: 7----->", <-chans[7])
	//fmt.Println("i: 8----->", <-chans[8])
	//fmt.Println("i: 9----->", <-chans[9])
	//fmt.Println(time.Since(t))
	//time.Sleep(2*time.Second)Ò

	s := []byte("*hello\r\n")
	newline := bytes.IndexByte(s, '\r')
	pos := newline + 2
	fmt.Println(string(s[pos:]))

}

//func (d *data) getCount(ch chan int64, i int) {
//	d.Lock()
//	count := d.count
//	fmt.Println("i:", i, "----->", count)
//	d.count++
//	d.Unlock()
//	ch <- count
//}
