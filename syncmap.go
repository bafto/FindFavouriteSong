package main

import "sync"

type SyncMap[K, V any] struct {
	m sync.Map
}

func (m *SyncMap[K, V]) Load(key K) (V, bool) {
	var defaultValue V

	v, ok := m.m.Load(key)
	if ok {
		return v.(V), ok
	}
	return defaultValue, ok
}

func (m *SyncMap[K, V]) Store(key K, value V) {
	m.m.Store(key, value)
}

func (m *SyncMap[K, V]) Delete(key K) {
	m.m.Delete(key)
}
