package cache

import "testing"

func TestLinkedListEmpty(t *testing.T) {
	l := newLinkedList[int]()
	if got := l.Back(); got != nil {
		t.Fatalf("Back() on empty list = %v, want nil", got)
	}
	if l.len != 0 {
		t.Fatalf("len = %d, want 0", l.len)
	}
}

func TestLinkedListPushFrontBackAndPrev(t *testing.T) {
	l := newLinkedList[int]()

	e1 := l.PushFront(1)
	e2 := l.PushFront(2)
	e3 := l.PushFront(3)

	if l.len != 3 {
		t.Fatalf("len = %d, want 3", l.len)
	}
	if got := l.Back(); got != e1 {
		t.Fatalf("Back() = %p, want %p", got, e1)
	}
	if got := e1.Prev(); got != e2 {
		t.Fatalf("e1.Prev() = %p, want %p", got, e2)
	}
	if got := e2.Prev(); got != e3 {
		t.Fatalf("e2.Prev() = %p, want %p", got, e3)
	}
	if got := e3.Prev(); got != nil {
		t.Fatalf("e3.Prev() = %p, want nil", got)
	}
}

func TestLinkedListMoveToFront(t *testing.T) {
	l := newLinkedList[int]()

	e1 := l.PushFront(1) // [1]
	e2 := l.PushFront(2) // [2 1]
	e3 := l.PushFront(3) // [3 2 1]

	l.MoveToFront(e1) // [1 3 2]

	if got := l.Back(); got != e2 {
		t.Fatalf("Back() = %p, want %p", got, e2)
	}
	if got := e2.Prev(); got != e3 {
		t.Fatalf("e2.Prev() = %p, want %p", got, e3)
	}
	if got := e3.Prev(); got != e1 {
		t.Fatalf("e3.Prev() = %p, want %p", got, e1)
	}
	if got := e1.Prev(); got != nil {
		t.Fatalf("e1.Prev() = %p, want nil", got)
	}
}

func TestLinkedListRemove(t *testing.T) {
	l := newLinkedList[int]()
	e1 := l.PushFront(1) // [1]
	e2 := l.PushFront(2) // [2 1]
	e3 := l.PushFront(3) // [3 2 1]

	l.Remove(e2) // [3 1]
	if l.len != 2 {
		t.Fatalf("len = %d, want 2", l.len)
	}
	if got := l.Back(); got != e1 {
		t.Fatalf("Back() = %p, want %p", got, e1)
	}
	if got := e1.Prev(); got != e3 {
		t.Fatalf("e1.Prev() = %p, want %p", got, e3)
	}
	if got := e3.Prev(); got != nil {
		t.Fatalf("e3.Prev() = %p, want nil", got)
	}
	if e2.list != nil || e2.prev != nil || e2.next != nil {
		t.Fatal("removed element should be detached from list")
	}
}

func TestLinkedListIgnoreInvalidElementOperations(t *testing.T) {
	l1 := newLinkedList[int]()
	l2 := newLinkedList[int]()
	e := l1.PushFront(1)

	// foreign element should be ignored
	l2.Remove(e)
	if l1.len != 1 {
		t.Fatalf("len after foreign remove = %d, want 1", l1.len)
	}
	l2.MoveToFront(e)
	if l1.len != 1 {
		t.Fatalf("len after foreign move = %d, want 1", l1.len)
	}

	// nil should be ignored
	l1.Remove(nil)
	l1.MoveToFront(nil)
	if l1.len != 1 {
		t.Fatalf("len after nil operations = %d, want 1", l1.len)
	}
}
