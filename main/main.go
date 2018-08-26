package main

import (
	"fmt"
	"sync"
	"time"
)

type data struct {
	count int64
	sync.Mutex
}

func main() {
	chans := make([]chan int64, 10)
	t := time.Now()
	d := data{
		0,
		sync.Mutex{},
	}
	for i:=0;i<10;i++ {
		chans[i] = make(chan int64)
		go d.getCount(chans[i], i)
	}
	fmt.Println("i: 0----->", <-chans[0])
	fmt.Println("i: 1----->", <-chans[1])
	fmt.Println("i: 2----->", <-chans[2])
	fmt.Println("i: 3----->", <-chans[3])
	fmt.Println("i: 4----->", <-chans[4])
	fmt.Println("i: 5----->", <-chans[5])
	fmt.Println("i: 6----->", <-chans[6])
	fmt.Println("i: 7----->", <-chans[7])
	fmt.Println("i: 8----->", <-chans[8])
	fmt.Println("i: 9----->", <-chans[9])
	fmt.Println(time.Since(t))
	time.Sleep(2*time.Second)
}

func (d *data) getCount(ch chan int64, i int) {
	d.Lock()
	count := d.count
	fmt.Println("i:", i, "----->", count)
	d.count++
	d.Unlock()
	ch <- count
}






