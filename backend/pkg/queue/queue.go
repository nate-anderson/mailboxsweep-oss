package queue

import "fmt"

type Queue[T any] struct {
	assigned bool
	Value    T
	Next     *Queue[T]
}

func (q *Queue[T]) tail() *Queue[T] {
	if q == nil {
		return nil
	}

	cur := q
	for cur.Next != nil {
		cur = cur.Next
	}
	return cur
}

func (q *Queue[T]) Enqueue(value T) {
	if !q.assigned {
		q.Value = value
		q.assigned = true
		return
	}

	t := q.tail()
	t.Next = &Queue[T]{
		Value:    value,
		assigned: true,
	}
}

func (q *Queue[T]) Dequeue() (T, bool) {
	if q == nil || !q.assigned {
		return *new(T), false
	}
	val := q.Value

	if q.Next == nil {
		*q = Queue[T]{}
	} else {
		*q = *q.Next
	}
	return val, true
}

func (q *Queue[T]) Len() int {
	i := 0
	for q != nil {
		i++
		q = q.Next
	}
	return i
}

func (q *Queue[T]) String() string {
	if q == nil {
		return "empty queue (nil)"
	}

	cur := q
	str := fmt.Sprintf("<%v> ", cur.Value)
	for cur.Next != nil {
		cur = cur.Next
		str += fmt.Sprintf("<%v> ", cur.Value)
	}

	return str
}
