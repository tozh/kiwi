package structure

import (
	"sync"
	"sync/atomic"
)

/* definition of List structure */

type ListNode struct {
	Prev  *ListNode
	Next  *ListNode
	Value interface{}
}

type ListIterator struct {
	node      *ListNode
	direction int
	start     *ListNode
	end       *ListNode
}

func (iter *ListIterator) RewindLeft(list *List) {
	iter.node = list.Left
	iter.direction = ITERATION_DIRECTION_INORDER
}

func (iter *ListIterator) RewindRight(list *List) {
	iter.node = list.Right
	iter.direction = ITERATION_DIRECTION_REVERSE_ORDER
}

func (iter *ListIterator) Next() (*ListNode) {
	if iter.direction >=0 {
		iter.node = iter.node.Next
		return iter.node
	} else {
		iter.node = iter.node.Prev
		return iter.node

	}
}

func (iter *ListIterator) HasNext() bool {
	return iter.node != iter.end
}

/* methods for ListIterator */
func (list *List) Iterator(direction int) *ListIterator {
	if direction >=0 {
		return &ListIterator{
			list.Left,
			direction,
			list.Left,
			list.Right,
		}
	} else {
		return &ListIterator{
			list.Right,
			direction,
			list.Right,
			list.Left,
		}
	}

}

type List struct {
	Left      *ListNode
	Right     *ListNode
	len       int32
	NodeEqual func(value interface{}, key interface{}) bool
	lLock     sync.RWMutex
	rLock     sync.RWMutex
}

func (list *List) Len() int32 {
	return atomic.LoadInt32(&list.len)
}

func (list *List) LenAdd(n int32) {
	atomic.AddInt32(&list.len, n)
}

func (list *List) LenMinus(n int32) {
	atomic.AddInt32(&list.len, -n)
}

func (list *List) LenSet(n int32) {
	atomic.StoreInt32(&list.len, n)
}

func (list *List) LeftAppend(value interface{}) {
	node := ListNode{
		nil,
		nil,
		value,
	}
	if list.Len() <= 3 {
		list.rLock.Lock()
		list.lLock.Lock()
		defer list.lLock.Unlock()
		defer list.rLock.Unlock()
	} else {
		list.lLock.Lock()
		defer list.lLock.Unlock()
	}
	node.Next = list.Left.Next
	list.Left.Next.Prev = &node
	node.Prev = list.Left
	list.Left.Next = &node
	list.LenAdd(1)
}

func (list *List) LeftPop() interface{} {
	if list.Len() == 0 {
		return nil
	}
	var node *ListNode
	if list.Len() <= 3 {
		list.rLock.Lock()
		list.lLock.Lock()
		defer list.lLock.Unlock()
		defer list.rLock.Unlock()
	} else {
		list.lLock.Lock()
		defer list.lLock.Unlock()
	}
	node = list.Left.Next
	node.Next.Prev = list.Left
	list.Left.Next = node.Next
	node.Next = nil
	node.Prev = nil
	list.LenMinus(1)
	return node.Value

}

func (list *List) Append(value interface{}) {
	node := ListNode{
		nil,
		nil,
		value,
	}
	if list.Len() <= 3 {
		list.rLock.Lock()
		list.lLock.Lock()
		defer list.lLock.Unlock()
		defer list.rLock.Unlock()
	} else {
		list.rLock.Lock()
		defer list.rLock.Unlock()
	}
	node.Prev = list.Right.Prev
	list.Right.Prev.Next = &node
	node.Next = list.Right
	list.Right.Prev = &node
	list.LenAdd(1)
}

func (list *List) Pop() interface{} {
	if list.Len() == 0 {
		return nil
	}
	var node *ListNode
	if list.Len() <= 3 {
		list.rLock.Lock()
		list.lLock.Lock()
		defer list.lLock.Unlock()
		defer list.rLock.Unlock()

	} else {
		list.rLock.Lock()
		defer list.rLock.Unlock()

	}
	node = list.Right.Prev
	node.Prev.Next = list.Right
	list.Right.Prev = node.Prev
	node.Next = nil
	node.Prev = nil
	list.LenMinus(1)
	return node.Value

}

/* Remove all the elements from the List without destroying the List itself. */
func (list *List) Clear() {
	list.rLock.Lock()
	list.lLock.Lock()
	defer list.lLock.Unlock()
	defer list.rLock.Unlock()

	list.LenSet(0)
	list.Left = &ListNode{
		nil,
		nil,
		nil,
	}
	list.Right = &ListNode{
		nil,
		nil,
		nil,
	}
	list.Left.Next = list.Right
	list.Right.Prev = list.Left
}

func (list *List) InsertNode(node *ListNode, value interface{}, after bool) {
	newNode := ListNode{
		nil,
		nil,
		value,
	}
	list.rLock.Lock()
	list.lLock.Lock()
	defer list.lLock.Unlock()
	defer list.rLock.Unlock()

	if after {
		newNode.Next = node.Next
		newNode.Prev = node
		node.Next = &newNode
	} else {
		newNode.Prev = node.Prev
		newNode.Next = node
		node.Prev = &newNode
	}
	list.LenAdd(1)
}

func (list *List) RemoveNode(node *ListNode) {
	list.rLock.Lock()
	list.lLock.Lock()
	defer list.lLock.Unlock()
	defer list.rLock.Unlock()

	prev := node.Prev
	next := node.Next

	prev.Next = next
	next.Prev = prev

	node.Prev = nil
	node.Next = nil
	list.LenMinus(1)
}

/* Search the list for a node matching a given key.
 * The NodeEqual is performed using the 'NodeEqual' method
 * set with listSetMatchMethod(). If no 'NodeEqual' method
 * is set, the 'Value' pointer of every node is directly
 * compared with the 'key' pointer.
 *
 * On success the first matching node pointer is returned
 * (search starts from Left). If no matching node exists
 * NULL is returned. */
func (list *List) SearchValue(val interface{}) (node *ListNode, index int) {
	list.rLock.RLock()
	list.lLock.RLock()
	defer list.lLock.RUnlock()
	defer list.rLock.RUnlock()
	index = 0

	iter := list.Iterator(ITERATION_DIRECTION_INORDER)
	for node = iter.Next(); iter.HasNext(); node = iter.Next() {
		if list.NodeEqual(val, node.Value) {
			return node, index
		}
		index++
	}
	return nil, -1
}

func (list *List) RSearchValue(val interface{}) (node *ListNode, index int) {
	list.rLock.RLock()
	list.lLock.RLock()
	defer list.lLock.RUnlock()
	defer list.rLock.RUnlock()
	index = 0
	iter := list.Iterator(ITERATION_DIRECTION_REVERSE_ORDER)
	for node = iter.Next(); iter.HasNext(); node = iter.Next() {
		if list.NodeEqual(val, node.Value) {
			return node, index
		}
		index++
	}
	return nil, -1
}

/* Return the element at the specified zero-based index
 * where 0 is the Left, 1 is the element next to Left
 * and so on. Negative integers are used in order to count
 * from the Right, -1 is the last element, -2 the penultimate
 * and so on. If the index is out of range NULL is returned. */
func (list *List) Index(index int) *ListNode {
	list.rLock.RLock()
	list.lLock.RLock()
	defer list.lLock.RUnlock()
	defer list.rLock.RUnlock()
	var node *ListNode = nil
	if index >= 0 && index < int(list.Len()) {
		if index >= int(list.Len())/2 {
			idx := int(list.Len()) - 1 - index
			node = list.Right
			for idx >= 0 {
				node = node.Prev
				idx--
			}
		} else {
			idx := index
			node = list.Left
			for idx >= 0 {
				node = node.Next
				idx--
			}
		}
	} else if index < 0 && -index <= int(list.Len()) {
		if -index >= int(list.Len())/2 {
			idx := int(list.Len()) + index
			node = list.Left
			for idx >= 0 {
				node = node.Next
				idx--
			}
		} else {
			idx := -index - 1
			node = list.Right
			for idx >= 0 {
				node = node.Prev
				idx--
			}
		}
	}
	return node

}

func (list *List) RotateLeft() {
	list.rLock.Lock()
	list.lLock.Lock()
	defer list.lLock.Unlock()
	defer list.rLock.Unlock()

	if list.Len() <= 1 {
		return
	}

	l := list.Left.Next
	r := list.Right.Prev
	list.Left.Next = l.Next
	l.Next.Prev = list.Left

	r.Next = l
	l.Prev = r
	l.Next = list.Right
	list.Right.Prev = l
}

func (list *List) RotateRight() {
	list.rLock.Lock()
	list.lLock.Lock()
	defer list.lLock.Unlock()
	defer list.rLock.Unlock()
	if list.Len() <= 1 {
		return
	}
	l := list.Left.Next
	r := list.Right.Prev
	list.Right.Prev = r.Prev
	r.Prev.Next = list.Right

	l.Prev = r
	r.Next = l
	r.Prev = list.Left
	list.Left.Next = r
}

func (list *List) LeftFirst() *ListNode {
	if list.Len() == 0 {
		return nil
	} else {
		return list.Left.Next
	}
}

func (list *List) RightFirst() *ListNode {
	if list.Len() == 0 {
		return nil
	} else {
		return list.Right.Prev
	}
}

/* join <other list> to the end of the list */
func (list *List) Join(other *List) {
	list.rLock.Lock()
	list.lLock.Lock()
	other.rLock.Lock()
	other.lLock.Lock()

	list.Right.Prev.Next = other.Left.Next
	list.Right = other.Right
	list.LenAdd(other.Len())

	other.lLock.Unlock()
	other.rLock.Unlock()
	list.lLock.Unlock()
	list.rLock.Unlock()
	other.Clear()
}

/* functions for copy List, but the value did not copy, this is a shallow copy*/
func ListCopy(list *List) *List {
	cp := ListCreate()
	cp.NodeEqual = list.NodeEqual
	iter := list.Iterator(ITERATION_DIRECTION_INORDER)
	for node := iter.Next(); iter.HasNext(); node = iter.Next() {
		cp.Append(node.Value)
	}
	return cp
}

func ListCreate() *List {
	list := List{
		Left: &ListNode{
			nil,
			nil,
			nil,

		},
		Right: &ListNode{
			nil,
			nil,
			nil,
		},
		len:       0,
		NodeEqual: nil,
		lLock:     sync.RWMutex{},
		rLock:     sync.RWMutex{},
	}
	list.Left.Next = list.Right
	list.Right.Prev = list.Left
	return &list
}