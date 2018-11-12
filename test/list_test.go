package test

import (
	"fmt"
	"time"
	"sync"
	"math/rand"
	"kiwi/src/structure"
)

func list_test() {
	AppendPopTest()
	SearchTest()
	IndexTest()
	RotateTest()
	JoinTest()
	CopyTest()
}

func AppendPopTest() {
	fmt.Println("AppendPopTest start")
	t1 := time.Now()
	list := structure.ListCreate()
	wg := sync.WaitGroup{}
	for i := 0; i < 500000; i++ {
		n := i
		if rand.Int()&1 == 0 {
			wg.Add(1)
			go func() {
				list.LeftAppend(&n)
				wg.Done()
			}()
		} else {
			wg.Add(1)
			go func() {
				list.Append(&n)
				wg.Done()
			}()
		}
	}

	for i := 0; i < 250000; i++ {
		n := i
		if n&1 == 0 {
			wg.Add(1)
			go func() {
				list.LeftPop()
				wg.Done()
			}()
		} else {
			wg.Add(1)
			go func() {
				list.Pop()
				wg.Done()
			}()
		}
	}
	wg.Wait()
	fmt.Println(list.Len())
	fmt.Printf("time is %v\n", time.Since(t1))

	bucket := [500000]int{}
	idx := 0
	iter := list.Iterator(structure.ITERATION_DIRECTION_INORDER)

	for node := iter.Next(); iter.HasNext(); node = iter.Next() {
		bucket[*(node.Value.(*int))]++
		idx++
	}
	count := 0
	for j := 0; j < len(bucket); j++ {
		if bucket[j] != 1 {
			count++
		}
	}

	fmt.Println("AppendPopTest finish, count:", count)
	fmt.Println()
}
func SearchTest() {
	fmt.Println("SearchTest start")
	t1 := time.Now()
	list := structure.ListCreate()
	listLen := 10000
	for i := 0; i < listLen; i++ {
		n := i
		list.Append(&n)
	}

	list.NodeEqual = func(node interface{}, other interface{}) bool {
		return *node.(*int) == *other.(*int)
	}

	for val := 0; val < listLen; val++ {
		node, idx := list.SearchValue(&val)
		rnode, ridx := list.RSearchValue(&val)
		if idx+ridx != listLen-1 || node != rnode {
			panic(fmt.Sprintf("Error Index. idx=%d, ridx=%d, node.val=%d, rnode.val=%d\n", idx, ridx, *node.Value.(*int), *rnode.Value.(*int)))
		}
	}

	fmt.Printf("SearchValue finish.time is %v\n", time.Since(t1))
	fmt.Println()
}

func IndexTest() {
	fmt.Println("IndexTest start")
	t1 := time.Now()
	list := structure.ListCreate()
	listLen := 50000

	for i := 0; i < listLen; i++ {
		n := i
		list.Append(&n)
	}

	list.NodeEqual = func(node interface{}, other interface{}) bool {
		return *node.(*int) == *other.(*int)
	}
	for i := 0; i < listLen; i++ {
		node := list.Index(i)
		if *node.Value.(*int) != i {
			panic(fmt.Sprintf("Error Index. idx=%d, node.val=%d\n", i, *node.Value.(*int)))
		}
		ri := i - listLen
		rnode := list.Index(ri)
		if *node.Value.(*int) != i {
			panic(fmt.Sprintf("Error RIndex. idx=%d, node.val=%d\n", i, *rnode.Value.(*int)))
		}
	}

	fmt.Printf("IndexTest finish. time is %v\n", time.Since(t1))
	fmt.Println()
}

func RotateTest() {
	fmt.Println("RotateTest start")
	t1 := time.Now()
	list := structure.ListCreate()
	listLen := 5000
	for i := 0; i < listLen; i++ {
		n := i
		list.Append(&n)
	}

	list.NodeEqual = func(node interface{}, other interface{}) bool {
		return *node.(*int) == *other.(*int)
	}
	for i := 3*listLen; i >=0; i-- {
		if i%listLen != *list.LeftFirst().Value.(*int) {
			panic(fmt.Sprintf("Error RotateRight. val=%d, node.val=%d\n", i%listLen, *list.LeftFirst().Value.(*int)))
		}
		list.RotateRight()
	}
	list.Clear()
	for i := 0; i < listLen; i++ {
		n := i
		list.Append(&n)
	}

	for i := 0; i < 3*listLen; i++ {
		if i%listLen != *list.LeftFirst().Value.(*int) {
			panic(fmt.Sprintf("Error RotateLeft. val=%d, node.val=%d\n", i%listLen, *list.LeftFirst().Value.(*int)))
		}
		list.RotateLeft()
	}

	fmt.Printf("RotateTest finish. time is %v\n", time.Since(t1))
	fmt.Println()
}


func JoinTest() {
	fmt.Println("JoinTest start")
	t1 := time.Now()
	list1 := structure.ListCreate()
	listLen := 5000
	for i := 0; i < listLen; i++ {
		n := i
		list1.Append(&n)
	}
	list2 := structure.ListCreate()
	for i := 0; i< listLen;i++ {
		n := i + listLen
		list2.Append(&n)
	}
	l1len := list1.Len()
	l2len := list2.Len()
	list1.Join(list2)
	idx := 0
	if list2.Len() != 0 {
		panic(fmt.Sprintf("Error JoinTest. list2.Left=%v, list2.Right=%v, list2.Len=%d\n", list2.Left, list2.Right, list2.Len()))
	}
	iter := list1.Iterator(structure.ITERATION_DIRECTION_INORDER)
	for node := iter.Next(); iter.HasNext(); node = iter.Next() {
		if node ==nil {
			fmt.Println(idx, node.Value)
		}
		if idx != *node.Value.(*int) {
			panic(fmt.Sprintf("Error JoinTest. idx=%d, node.val=%d\n", idx, *node.Value.(*int)))
		}
		idx++
	}
	if idx != int(list1.Len()) || list1.Len() != l1len+l2len {
		panic(fmt.Sprintf("Error JoinTest. idx=%d, list1.Len()=%d, l1len=%d, l2len=%d\n", idx, list1.Len(), l1len, l2len))
	}


	fmt.Printf("RotateTest finish. time is %v\n", time.Since(t1))
	fmt.Println()
}

func CopyTest() {
	fmt.Println("CopyTest start")
	t1 := time.Now()
	list := structure.ListCreate()
	listLen := 5000
	for i := 0; i < listLen; i++ {
		n := i
		list.Append(&n)
	}
	list_copy := structure.ListCopy(list)

	idx := 0
	iter := list_copy.Iterator(1)
	//fmt.Println(iter.Next(), iter.Next())

	for node := iter.Next(); iter.HasNext(); node = iter.Next() {
		if idx != *node.Value.(*int) {
			panic(fmt.Sprintf("CopyTest Error, idx=%d, node.val=%d\n", idx, *node.Value.(*int)))
		}
		idx++
	}


	fmt.Printf("CopyTest finish. time is %v\n", time.Since(t1))
	fmt.Println()
}
