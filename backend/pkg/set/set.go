package set

type Set[T comparable] struct {
	vals map[T]struct{}
}

func NewSet[T comparable](vals ...T) Set[T] {
	s := Set[T]{
		vals: make(map[T]struct{}),
	}
	for _, v := range vals {
		s.Add(v)
	}
	return s
}

// returns false if item was already contained, true if it was added
func (s *Set[T]) Add(v T) bool {
	if _, ok := s.vals[v]; ok {
		return false
	}
	s.vals[v] = struct{}{}
	return true
}

func (s *Set[T]) Has(v T) bool {
	_, ok := s.vals[v]
	return ok
}

func (s Set[T]) Slice() []T {
	sl := make([]T, 0, len(s.vals))
	for k := range s.vals {
		sl = append(sl, k)
	}
	return sl
}

// returns true if the item was removed, false if it wasn't contained
func (s *Set[T]) Remove(v T) bool {
	if _, ok := s.vals[v]; !ok {
		return false
	}
	delete(s.vals, v)
	return true
}

func (s *Set[T]) Len() int {
	return len(s.vals)
}
