package server

import (
	"sync"
)

type SyncList struct {
	Head  *ListNode
	Tail  *ListNode
	Len   int64
	Match func(value interface{}, key interface{}) bool
	mutex sync.RWMutex
}

/* SyncList methods */
func (list *SyncList) ListLength() int64 {
	list.mutex.RLock()
	defer list.mutex.RUnlock()
	return list.Len
}

func (list *SyncList) ListHead() *ListNode {
	list.mutex.RLock()
	defer list.mutex.RUnlock()
	return list.Head
}

func (list *SyncList) ListTail() *ListNode {
	list.mutex.RLock()
	defer list.mutex.RUnlock()
	return list.Tail
}

func (list *SyncList) ListSetMatch(match func(value interface{}, key interface{}) bool) {
	list.mutex.Lock()
	defer list.mutex.Unlock()
	list.Match = match
}

func (list *SyncList) ListAddNodeHead(value interface{}) {
	list.mutex.Lock()
	defer list.mutex.Unlock()
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

func (list *SyncList) ListAddNodeTail(value interface{}) {
	list.mutex.Lock()
	defer list.mutex.Unlock()
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

/* Remove all the elements from the SyncList without destroying the SyncList itself. */
func (list *SyncList) ListEmpty() {
	list.mutex.Lock()
	defer list.mutex.Unlock()
	list.Len = 0
	list.Head, list.Tail = nil, nil
}

func (list *SyncList) ListInsertNode(oldNode *ListNode, value interface{}, after bool) {
	list.mutex.Lock()
	defer list.mutex.Unlock()
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

func (list *SyncList) ListDelNode(node *ListNode) {
	list.mutex.Lock()
	defer list.mutex.Unlock()
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
func (list *SyncList) ListSearchKey(key interface{}) *ListNode {
	list.mutex.RLock()
	defer list.mutex.RUnlock()
	iter := list.ListGetIterator(ITERATION_DIRECTION_INORDER)
	for node := iter.ListNext(); node != nil; node = iter.ListNext() {
		if list.Match != nil {
			if list.Match(node.Value, key) {
				return node
			}
		} else {
			if key == node.Value {
				return node
			}
		}
	}
	return nil
}

/* Return the element at the specified zero-based index
 * where 0 is the Head, 1 is the element Next to Head
 * and so on. Negative integers are used in order to count
 * from the Tail, -1 is the last element, -2 the penultimate
 * and so on. If the index is out of range NULL is returned. */
func (list *SyncList) ListIndex(index int) *ListNode {
	list.mutex.RLock()
	defer list.mutex.RUnlock()
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

func (list *SyncList) ListRotate() {
	list.mutex.Lock()
	defer list.mutex.Unlock()
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
func (list *SyncList) ListJoin(other *SyncList) {
	list.mutex.Lock()
	defer list.mutex.Unlock()
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
func (list *SyncList) ListGetIterator(direction int) *listIter {
	list.mutex.RLock()
	defer list.mutex.RUnlock()
	iter := listIter{}
	if direction == ITERATION_DIRECTION_INORDER {
		iter.Next = list.Head
	} else {
		iter.Next = list.Tail
	}
	iter.Direction = direction
	return &iter
}

/* functions for SyncList, for Create, Dup*/
func ListDup(list *SyncList) *SyncList {
	cp := CreateSyncList()
	if cp == nil {
		return cp
	}
	cp.Match = list.Match

	iter := list.ListGetIterator(ITERATION_DIRECTION_INORDER)
	for node := iter.ListNext(); node != nil; node = iter.ListNext() {
		value := node.Value
		cp.ListAddNodeTail(value)
	}
	return cp
}

func CreateSyncList() *SyncList {
	return &SyncList{
		Head:  nil,
		Tail:  nil,
		Len:   0,
		Match: nil,
		mutex: sync.RWMutex{},
	}
}
