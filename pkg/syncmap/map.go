package syncmap

// Strongly typed, concurrency-safe map wrappers

import (
	"encoding/json"
	"sync"
)

type Map[keyT comparable, valueT any] struct {
	m   map[keyT]valueT
	mux sync.Mutex
}

func New[keyT comparable, valueT any]() Map[keyT, valueT] {
	return Map[keyT, valueT]{
		m:   make(map[keyT]valueT),
		mux: sync.Mutex{},
	}
}

func (m *Map[keyT, valueT]) Get(key keyT) (valueT, bool) {
	v, ok := m.m[key]
	return v, ok
}

func (m *Map[keyT, valueT]) Set(key keyT, value valueT) {
	m.mux.Lock()
	defer m.mux.Unlock()
	m.m[key] = value
}

func (m *Map[keyT, valueT]) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.m)
}

func (m *Map[keyT, valueT]) UnmarshalJSON(data []byte) error {
	m.mux.Lock()
	defer m.mux.Unlock()
	return json.Unmarshal(data, &m.m)
}
