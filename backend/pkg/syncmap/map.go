package syncmap

// Strongly typed, concurrency-safe map wrappers

import (
	"encoding/json"
	"fmt"
	"sync"
)

type Map[keyT comparable, valueT any] struct {
	m   map[keyT]valueT
	mux sync.RWMutex
}

func New[keyT comparable, valueT any]() Map[keyT, valueT] {
	return Map[keyT, valueT]{
		m:   make(map[keyT]valueT),
		mux: sync.RWMutex{},
	}
}

func (m *Map[keyT, valueT]) Get(key keyT) (valueT, bool) {
	m.mux.RLock()
	v, ok := m.m[key]
	m.mux.RUnlock()
	return v, ok
}

func (m *Map[keyT, valueT]) Set(key keyT, value valueT) {
	m.mux.Lock()
	m.m[key] = value
	m.mux.Unlock()
}

func (m *Map[keyT, valueT]) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.m)
}

func (m *Map[keyT, valueT]) UnmarshalJSON(data []byte) error {
	m.mux.Lock()
	err := json.Unmarshal(data, &m.m)
	m.mux.Unlock()
	return err
}

// use sparingly.
// iterate over the map's keys and values, with the ability to mutate it
func (m *Map[keyT, valueT]) RangeWrite(fn func(key keyT, value valueT)) {
	m.mux.Lock()
	for key, value := range m.m {
		fn(key, value)
	}
	m.mux.Unlock()
}

// use sparingly.
// iterate over the map's keys and values,
func (m *Map[keyT, valueT]) RangeRead(fn func(key keyT, value valueT)) {
	m.mux.RLock()
	for key, value := range m.m {
		fn(key, value)
	}
	m.mux.RUnlock()
}

func (m *Map[keyT, valueT]) Len() int {
	m.mux.RLock()
	l := len(m.m)
	m.mux.RUnlock()
	return l
}

func (m *Map[keyT, valueT]) String() string {
	m.mux.RLock()
	str := fmt.Sprintf("%+v", m.m)
	m.mux.RUnlock()
	return str
}

func (m *Map[keyT, valueT]) Copy() *Map[keyT, valueT] {
	newMap := make(map[keyT]valueT)
	m.mux.RLock()
	for key, value := range m.m {
		newMap[key] = value
	}
	m.mux.RUnlock()
	return &Map[keyT, valueT]{
		m:   newMap,
		mux: sync.RWMutex{},
	}
}

func (m *Map[keyT, valueT]) Map() map[keyT]valueT {
	m.mux.RLock()
	mmap := m.m
	m.mux.RUnlock()
	return mmap
}
