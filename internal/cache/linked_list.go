package cache

type linkedList[T any] struct {
	root listElement[T]
	len  int
}

type listElement[T any] struct {
	next *listElement[T]
	prev *listElement[T]
	list *linkedList[T]

	Value T
}

func newLinkedList[T any]() *linkedList[T] {
	l := &linkedList[T]{}
	l.root.next = &l.root
	l.root.prev = &l.root
	return l
}

func (l *linkedList[T]) Back() *listElement[T] {
	if l.len == 0 {
		return nil
	}
	return l.root.prev
}

func (l *linkedList[T]) PushFront(v T) *listElement[T] {
	return l.insertValue(v, &l.root)
}

func (l *linkedList[T]) MoveToFront(e *listElement[T]) {
	if e == nil || e.list != l || l.root.next == e {
		return
	}
	l.move(e, &l.root)
}

func (l *linkedList[T]) Remove(e *listElement[T]) {
	if e == nil || e.list != l {
		return
	}
	l.remove(e)
}

func (e *listElement[T]) Prev() *listElement[T] {
	if e == nil || e.list == nil {
		return nil
	}
	p := e.prev
	if p == &e.list.root {
		return nil
	}
	return p
}

func (l *linkedList[T]) insertValue(v T, at *listElement[T]) *listElement[T] {
	e := &listElement[T]{Value: v}
	return l.insert(e, at)
}

func (l *linkedList[T]) insert(e, at *listElement[T]) *listElement[T] {
	n := at.next
	at.next = e
	e.prev = at
	e.next = n
	n.prev = e
	e.list = l
	l.len++
	return e
}

func (l *linkedList[T]) remove(e *listElement[T]) {
	e.prev.next = e.next
	e.next.prev = e.prev
	e.next = nil
	e.prev = nil
	e.list = nil
	l.len--
}

func (l *linkedList[T]) move(e, at *listElement[T]) {
	if e == at {
		return
	}
	e.prev.next = e.next
	e.next.prev = e.prev

	n := at.next
	at.next = e
	e.prev = at
	e.next = n
	n.prev = e
}
