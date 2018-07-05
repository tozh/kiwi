package src

/* definition of List struct */
type listNode struct{
	prev *listNode
	next *listNode
	value interface{}
}

/* listNode methods */
func (node *listNode) ListPrevNode() *listNode{
	return node.prev
}

func (node *listNode) ListNextNode() *listNode{
	return node.next
}

func (node *listNode) ListNodeValue() interface{} {
	return node.value
}

type listIter struct{
	next *listNode
	direction int
}

func (iter *listIter) ListNext() *listNode {
	current := iter.next
	if current != nil {
		if iter.direction == ITERATION_DIRECTION_INORDER {
			iter.next = current.next
		} else {
			iter.next = current.prev
		}
	}
	return current
}

func (iter *listIter) ListRewind(list *List) {
	iter.next = list.head
	iter.direction = ITERATION_DIRECTION_INORDER
}

func (iter *listIter) ListRewindTail(list *List) {
	iter.next = list.tail
	iter.direction = ITERATION_DIRECTION_REVERSE_ORDER
}


type List struct{
	head *listNode
	tail *listNode
	len int
	match func(value interface{}, key interface{}) bool
}

/* List methods */
func (list *List) ListLength() int{
	return list.len
}

func (list *List) ListHead() *listNode{
	return list.head
}

func (list *List) ListTail() *listNode{
	return list.tail
}


func (list *List) ListSetMatch(match func(value interface{}, key interface{}) bool) {
	list.match = match
}

func (list *List) ListAddNodeHead(value interface{}) {
	node := listNode{}
	node.value = value
	if list.len == 0 {
		list.head = &node
		list.tail = &node
		node.prev = nil
		node.next = nil
	} else {
		list.head.prev = &node
		node.next = list.head
		list.head = &node
		node.prev = nil
	}
	list.len++
}

func (list *List) ListAddNodeTail(value interface{}) {
	node := listNode{}
	node.value = value
	if list.len == 0 {
		list.head = &node
		list.tail = &node
		node.prev = nil
		node.next = nil
	} else {
		list.tail.next = &node
		node.prev = list.tail
		list.tail = &node
		node.next = nil
	}
	list.len++
}

/* Remove all the elements from the List without destroying the List itself. */
func (list *List) ListEmpty() {
	list.len = 0
	list.head, list.tail = nil, nil
}

func (list *List) ListInsertNode(oldNode *listNode, value interface{}, after bool) {
	node := listNode{}
	node.value = value
	if after {
		node.prev = oldNode
		node.next = oldNode.next
		if oldNode==list.tail {
			list.tail = &node
		}
	} else {
		node.next = oldNode
		node.prev = oldNode.prev
		if list.head == oldNode {
			list.head = &node
		}
	}
	if node.prev != nil {
		node.prev.next = &node
	}
	if node.next != nil {
		node.next.prev = &node
	}
	list.len++
}

func (list *List) ListDelNode(node *listNode){
	if node.prev != nil {
		node.prev.next = node.next
	} else {
		list.head = node.next
	}
	if node.next != nil {
		node.next.prev = node.prev
	} else {
		list.tail = node.prev
	}
	list.len--
}

/* Search the list for a node matching a given key.
 * The match is performed using the 'match' method
 * set with listSetMatchMethod(). If no 'match' method
 * is set, the 'value' pointer of every node is directly
 * compared with the 'key' pointer.
 *
 * On success the first matching node pointer is returned
 * (search starts from head). If no matching node exists
 * NULL is returned. */
func (list *List) ListSearchKey(key interface{}) *listNode {
	iter := list.ListGetIterator(ITERATION_DIRECTION_INORDER)
	node := iter.ListNext()

	for node!=nil {
		if list.match != nil {
			if list.match(node.value, key) {
				return node
			}
		}else {
			if key == node.value {
				return node
			}
		}
		node = iter.ListNext()
	}
	return nil
}

/* Return the element at the specified zero-based index
 * where 0 is the head, 1 is the element next to head
 * and so on. Negative integers are used in order to count
 * from the tail, -1 is the last element, -2 the penultimate
 * and so on. If the index is out of range NULL is returned. */
func (list *List) ListIndex(index int) *listNode {
	node := listNode{}
	if index<0 {
		index = (-index) - 1
		node := list.tail
		for index>=0 && node!=nil {
			node = node.prev
			index--
		}
	}else {
		node := list.head
		for index>=0 && node!=nil {
			node = node.next
			index--
		}
	}
	return &node
}

func (list *List) ListRotate() {
	if list.ListLength()<=1 {
		return
	}
	tail := list.tail
	/* detach current tail */
	list.tail = tail.prev
	list.tail.next = nil

	/* move tail to the head */
	list.head.prev = tail
	list.tail.prev = nil
	tail.next = list.head
	list.head = tail
}

/* join <other list> to the end of the list */
func (list *List) ListJoin(other *List) {
	if other.head != nil {
		other.head.prev = list.tail
	}

	if list.tail != nil {
		list.tail.next = other.head
	} else {
		list.head = other.head
	}
	if other.tail != nil {
		list.tail = other.tail
	}
	list.len += other.len

	/* Setup other as an empty list. */
	other.ListEmpty()
}

/* methods for listIter */
func (list *List) ListGetIterator(direction int) *listIter{
	iter := listIter{}
	if direction == ITERATION_DIRECTION_INORDER {
		iter.next = list.head
	} else {
		iter.next = list.tail
	}
	iter.direction = direction
	return &iter
}


/* functions for List, for Create, Dup*/
func ListDup(list *List) *List {
	cp := ListCreate()
	if cp == nil {
		return cp
	}
	cp.match = list.match
	iter := list.ListGetIterator(ITERATION_DIRECTION_INORDER)
	node:= iter.ListNext()
	for node!=nil {
		value := node.value
		cp.ListAddNodeTail(value)
		node = iter.ListNext()
	}
	return cp
}

func ListCreate() *List {
	list := List{}
	list.head, list.tail = nil, nil
	list.len = 0
	list.match = nil
	return &list
}

