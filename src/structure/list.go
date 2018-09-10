package structure

import . "redigo/src/constant"

/* definition of List struct */
type ListNode struct {
	Prev  *ListNode
	Next  *ListNode
	Value interface{}
}

/* ListNode methods */
func (node *ListNode) ListPrevNode() *ListNode {
	return node.Prev
}

func (node *ListNode) ListNextNode() *ListNode {
	return node.Next
}

func (node *ListNode) ListNodeValue() interface{} {
	return node.Value
}

type listIter struct {
	Next      *ListNode
	Direction int
}

func (iter *listIter) ListNext() *ListNode {
	current := iter.Next
	if current != nil {
		if iter.Direction == ITERATION_DIRECTION_INORDER {
			iter.Next = current.Next
		} else {
			iter.Next = current.Prev
		}
	}
	return current
}

func (iter *listIter) ListRewind(list *List) {
	iter.Next = list.Head
	iter.Direction = ITERATION_DIRECTION_INORDER
}

func (iter *listIter) ListRewindTail(list *List) {
	iter.Next = list.Tail
	iter.Direction = ITERATION_DIRECTION_REVERSE_ORDER
}

type List struct {
	Head  *ListNode
	Tail  *ListNode
	Len   int64
	Match func(value interface{}, key interface{}) bool
}

/* List methods */
func (list *List) ListLength() int64 {
	return list.Len
}

func (list *List) ListHead() *ListNode {
	return list.Head
}

func (list *List) ListTail() *ListNode {
	return list.Tail
}

func (list *List) ListSetMatch(match func(value interface{}, key interface{}) bool) {
	list.Match = match
}

func (list *List) ListAddNodeHead(value interface{}) {
	node := ListNode{}
	node.Value = value
	if list.Len == 0 {
		list.Head = &node
		list.Tail = &node
		node.Prev = nil
		node.Next = nil
	} else {
		list.Head.Prev = &node
		node.Next = list.Head
		list.Head = &node
		node.Prev = nil
	}
	list.Len++
}

func (list *List) ListAddNodeTail(value interface{}) {
	node := ListNode{}
	node.Value = value
	if list.Len == 0 {
		list.Head = &node
		list.Tail = &node
		node.Prev = nil
		node.Next = nil
	} else {
		list.Tail.Next = &node
		node.Prev = list.Tail
		list.Tail = &node
		node.Next = nil
	}
	list.Len++
}

/* Remove all the elements from the List without destroying the List itself. */
func (list *List) ListEmpty() {
	list.Len = 0
	list.Head, list.Tail = nil, nil
}

func (list *List) ListInsertNode(oldNode *ListNode, value interface{}, after bool) {
	node := ListNode{}
	node.Value = value
	if after {
		node.Prev = oldNode
		node.Next = oldNode.Next
		if oldNode == list.Tail {
			list.Tail = &node
		}
	} else {
		node.Next = oldNode
		node.Prev = oldNode.Prev
		if list.Head == oldNode {
			list.Head = &node
		}
	}
	if node.Prev != nil {
		node.Prev.Next = &node
	}
	if node.Next != nil {
		node.Next.Prev = &node
	}
	list.Len++
}

func (list *List) ListDelNode(node *ListNode) {
	if node.Prev != nil {
		node.Prev.Next = node.Next
	} else {
		list.Head = node.Next
	}
	if node.Next != nil {
		node.Next.Prev = node.Prev
	} else {
		list.Tail = node.Prev
	}
	list.Len--
}

/* Search the list for a node matching a given key.
 * The Match is performed using the 'Match' method
 * set with listSetMatchMethod(). If no 'Match' method
 * is set, the 'Value' pointer of every node is directly
 * compared with the 'key' pointer.
 *
 * On success the first matching node pointer is returned
 * (search starts from Head). If no matching node exists
 * NULL is returned. */
func (list *List) ListSearchKey(key interface{}) *ListNode {
	iter := list.ListGetIterator(ITERATION_DIRECTION_INORDER)
	node := iter.ListNext()

	for node != nil {
		if list.Match != nil {
			if list.Match(node.Value, key) {
				return node
			}
		} else {
			if key == node.Value {
				return node
			}
		}
		node = iter.ListNext()
	}
	return nil
}

/* Return the element at the specified zero-based index
 * where 0 is the Head, 1 is the element Next to Head
 * and so on. Negative integers are used in order to count
 * from the Tail, -1 is the last element, -2 the penultimate
 * and so on. If the index is out of range NULL is returned. */
func (list *List) ListIndex(index int) *ListNode {
	node := ListNode{}
	if index < 0 {
		index = (-index) - 1
		node := list.Tail
		for index >= 0 && node != nil {
			node = node.Prev
			index--
		}
	} else {
		node := list.Head
		for index >= 0 && node != nil {
			node = node.Next
			index--
		}
	}
	return &node
}

func (list *List) ListRotate() {
	if list.ListLength() <= 1 {
		return
	}
	tail := list.Tail
	/* detach current Tail */
	list.Tail = tail.Prev
	list.Tail.Next = nil

	/* move Tail to the Head */
	list.Head.Prev = tail
	list.Tail.Prev = nil
	tail.Next = list.Head
	list.Head = tail
}

/* join <other list> to the end of the list */
func (list *List) ListJoin(other *List) {
	if other.Head != nil {
		other.Head.Prev = list.Tail
	}

	if list.Tail != nil {
		list.Tail.Next = other.Head
	} else {
		list.Head = other.Head
	}
	if other.Tail != nil {
		list.Tail = other.Tail
	}
	list.Len += other.Len

	/* Setup other as an empty list. */
	other.ListEmpty()
}

/* methods for listIter */
func (list *List) ListGetIterator(direction int) *listIter {
	iter := listIter{}
	if direction == ITERATION_DIRECTION_INORDER {
		iter.Next = list.Head
	} else {
		iter.Next = list.Tail
	}
	iter.Direction = direction
	return &iter
}

/* functions for List, for Create, Dup*/
func ListDup(list *List) *List {
	cp := ListCreate()
	if cp == nil {
		return cp
	}
	cp.Match = list.Match
	iter := list.ListGetIterator(ITERATION_DIRECTION_INORDER)
	node := iter.ListNext()
	for node != nil {
		value := node.Value
		cp.ListAddNodeTail(value)
		node = iter.ListNext()
	}
	return cp
}

func ListCreate() *List {
	list := List{}
	list.Head, list.Tail = nil, nil
	list.Len = 0
	list.Match = nil
	return &list
}
