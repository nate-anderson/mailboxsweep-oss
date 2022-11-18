package queue

import (
	"fmt"
	"testing"
)

func TestQueue(t *testing.T) {
	q := Queue[int]{}
	var (
		first  = 10
		second = 20
		third  = 30
	)
	t.Run("enqueue", func(t *testing.T) {
		q.Enqueue(first)

		if q.Value != first {
			fmt.Println(q.String())
			t.Errorf("first value should equal %d: got %d", first, q.Value)
		}

		q.Enqueue(second)
		if q.Next.Value != second {
			t.Errorf("second value should equal %d: got %d", second, q.Value)
		}

		q.Enqueue(third)
		if q.Next.Next.Value != third {
			t.Errorf("third value should equal %d: got %d", third, q.Value)
		}
	})

	t.Run("len", func(t *testing.T) {
		if q.Len() != 3 {
			t.Errorf("len should equal 3 after 3 appends")
		}
	})

	t.Run("dequeue", func(t *testing.T) {
		l := 3
		vals := []int{first, second, third}
		for i := 0; i < l; i++ {
			if qLen := q.Len(); qLen != l-i {
				t.Errorf("expected len %d after %d dequeues: got len %d", l-i, i, qLen)
			}

			if val, ok := q.Dequeue(); val != vals[i] || !ok {
				t.Errorf("expected %dth dequeue to produce value %d: got %d", i, vals[i], val)
			}
		}
	})
}
