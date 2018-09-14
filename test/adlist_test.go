package test

import (
	. "KiwiDB/src/server"
	"fmt"
)

type data struct {
	name string
	age  int
}

func test() {
	list := CreateList()
	for i := 0; i < 10; i++ {
		d := data{
			"tong",
			i,
		}
		list.ListAddNodeTail(d)
	}
	iter := list.ListGetIterator(ITERATION_DIRECTION_INORDER)
	node := iter.ListNext()
	for node != nil {
		fmt.Println(node.ListNodeValue())
		node = iter.ListNext()
	}

	cpList := DupList(list)
	iter2 := cpList.ListGetIterator(ITERATION_DIRECTION_INORDER)
	node2 := iter2.ListNext()
	for node2 != nil {
		fmt.Println(node2.ListNodeValue())
		node2 = iter2.ListNext()
	}
}
