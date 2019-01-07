package structure

import (
	"sync"
	"sync/atomic"
)

/* definition of List structure */

type ListNode struct {
	prev  *ListNode
	next  *ListNode
	Value interface{}
}

type ListIterator struct {
	node      *ListNode
	direction int
	start     *ListNode
	end       *ListNode
}

func (iter *ListIterator) RewindLeft(list *List) {
	iter.node = list.l
	iter.direction = ITERATION_DIRECTION_INORDER
}

func (iter *ListIterator) RewindRight(list *List) {
	iter.node = list.r
	iter.direction = ITERATION_DIRECTION_REVERSE_ORDER
}

func (iter *ListIterator) Next() (*ListNode) {
	if iter.direction >= 0 {
		iter.node = iter.node.next
		return iter.node
	} else {
		iter.node = iter.node.prev
		return iter.node

	}
}

func (iter *ListIterator) HasNext() bool {
	return iter.node != iter.end
}

/* methods for ListIterator */
func (list *List) Iterator(direction int) *ListIterator {
	if direction >= 0 {
		return &ListIterator{
			list.l,
			direction,
			list.l,
			list.r,
		}
	} else {
		return &ListIterator{
			list.r,
			direction,
			list.r,
			list.l,
		}
	}

}

type List struct {
	l         *ListNode
	r         *ListNode
	len       int32
	NodeEqual func(value interface{}, key interface{}) bool
	lLock     sync.RWMutex
	rLock     sync.RWMutex
}

func (list *List) Len() int32 {
	return atomic.LoadInt32(&list.len)
}

func (list *List) lenAdd(n int32) {
	atomic.AddInt32(&list.len, n)
}

func (list *List) lenMinus(n int32) {
	atomic.AddInt32(&list.len, -n)
}

func (list *List) lenSet(n int32) {
	atomic.StoreInt32(&list.len, n)
}

func (list *List) Left() *ListNode {
	return list.l
}

func (list *List) Right() *ListNode {
	return list.r
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
	node.next = list.l.next
	list.l.next.prev = &node
	node.prev = list.l
	list.l.next = &node
	list.lenAdd(1)
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
	node = list.l.next
	node.next.prev = list.l
	list.l.next = node.next
	node.next = nil
	node.prev = nil
	list.lenMinus(1)
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
	node.prev = list.r.prev
	list.r.prev.next = &node
	node.next = list.r
	list.r.prev = &node
	list.lenAdd(1)
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
	node = list.r.prev
	node.prev.next = list.r
	list.r.prev = node.prev
	node.next = nil
	node.prev = nil
	list.lenMinus(1)
	return node.Value

}

/* Remove all the elements from the List without destroying the List itself. */
func (list *List) Clear() {
	list.rLock.Lock()
	list.lLock.Lock()
	defer list.lLock.Unlock()
	defer list.rLock.Unlock()

	list.lenSet(0)
	list.l = &ListNode{
		nil,
		nil,
		nil,
	}
	list.r = &ListNode{
		nil,
		nil,
		nil,
	}
	list.l.next = list.r
	list.r.prev = list.l
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
		newNode.next = node.next
		newNode.prev = node
		node.next = &newNode
	} else {
		newNode.prev = node.prev
		newNode.next = node
		node.prev = &newNode
	}
	list.lenAdd(1)
}

func (list *List) RemoveNode(node *ListNode) {
	list.rLock.Lock()
	list.lLock.Lock()
	defer list.lLock.Unlock()
	defer list.rLock.Unlock()

	prev := node.prev
	next := node.next

	prev.next = next
	next.prev = prev

	node.prev = nil
	node.next = nil
	list.lenMinus(1)
}

/* Search the list for a node matching a given key.
 * The NodeEqual is performed using the 'NodeEqual' method
 * set with listSetMatchMethod(). If no 'NodeEqual' method
 * is set, the 'Value' pointer of every node is directly
 * compared with the 'key' pointer.
 *
 * On success the first matching node pointer is returned
 * (search starts from l). If no matching node exists
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
 * where 0 is the l, 1 is the element next to l
 * and so on. Negative integers are used in order to count
 * from the r, -1 is the last element, -2 the penultimate
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
			node = list.r
			for idx >= 0 {
				node = node.prev
				idx--
			}
		} else {
			idx := index
			node = list.l
			for idx >= 0 {
				node = node.next
				idx--
			}
		}
	} else if index < 0 && -index <= int(list.Len()) {
		if -index >= int(list.Len())/2 {
			idx := int(list.Len()) + index
			node = list.l
			for idx >= 0 {
				node = node.next
				idx--
			}
		} else {
			idx := -index - 1
			node = list.r
			for idx >= 0 {
				node = node.prev
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

	l := list.l.next
	r := list.r.prev
	list.l.next = l.next
	l.next.prev = list.l

	r.next = l
	l.prev = r
	l.next = list.r
	list.r.prev = l
}

func (list *List) RotateRight() {
	list.rLock.Lock()
	list.lLock.Lock()
	defer list.lLock.Unlock()
	defer list.rLock.Unlock()
	if list.Len() <= 1 {
		return
	}
	l := list.l.next
	r := list.r.prev
	list.r.prev = r.prev
	r.prev.next = list.r

	l.prev = r
	r.next = l
	r.prev = list.l
	list.l.next = r
}

func (list *List) LeftFirst() *ListNode {
	if list.Len() == 0 {
		return nil
	} else {
		return list.l.next
	}
}

func (list *List) RightFirst() *ListNode {
	if list.Len() == 0 {
		return nil
	} else {
		return list.r.prev
	}
}

/* join <other list> to the end of the list */
func (list *List) Join(other *List) {
	list.rLock.Lock()
	list.lLock.Lock()
	other.rLock.Lock()
	other.lLock.Lock()

	list.r.prev.next = other.l.next
	list.r = other.r
	list.lenAdd(other.Len())

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
		l: &ListNode{
			nil,
			nil,
			nil,
		},
		r: &ListNode{
			nil,
			nil,
			nil,
		},
		len:       0,
		NodeEqual: nil,
		lLock:     sync.RWMutex{},
		rLock:     sync.RWMutex{},
	}
	list.l.next = list.r
	list.r.prev = list.l
	return &list
}
