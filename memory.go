package relevantcache

import (
	"bytes"
	"fmt"
	"sync"
	"time"
)

type memoryCacheEntry struct {
	data       []byte
	expiration time.Time
}

func (m memoryCacheEntry) Expired() bool {
	if m.expiration.IsZero() {
		return false
	}
	return m.expiration.After(time.Now())
}

type MemoryCache struct {
	data map[string]memoryCacheEntry
	mu   sync.Mutex
}

func NewMemoryCahe() *MemoryCache {
	return &MemoryCache{
		data: make(map[string]memoryCacheEntry),
	}
}

func (m *MemoryCache) Close() error {
	return nil
}

func (m *MemoryCache) Get(item interface{}) ([]byte, error) {
	key, err := getKey(item)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	entry, ok := m.data[key]
	if !ok {
		return nil, fmt.Errorf("record doesn't exist for key: %s", key)
	} else if entry.Expired() {
		delete(m.data, key)
		return nil, err
	}
	_, data := decodeMeta(entry.data)
	return data, nil
}

func (m *MemoryCache) Set(args ...interface{}) (err error) {
	var key string
	var value interface{}
	var ttl int

	switch len(args) {
	case 0:
		return fmt.Errorf("argments not enough")
	case 1:
		item, ok := args[0].(*Item)
		if !ok {
			return fmt.Errorf("if and only one argument is supplied, it must be *Item")
		}
		key = item.cacheKey()
		value = item.encode()
		ttl = int(item.ttl)
	case 2:
		key = args[0].(string)
		value = args[1]
		ttl = 0
	case 3:
		key = args[0].(string)
		value = args[1]
		ttl = args[2].(int)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	var expiration time.Time
	if ttl > 0 {
		expiration = time.Now().Add(time.Duration(ttl) * time.Second)
	}
	var dat []byte
	switch t := value.(type) {
	case string:
		dat = []byte(t)
	case []byte:
		dat = t
	}
	m.data[key] = memoryCacheEntry{
		data:       dat,
		expiration: expiration,
	}
	return nil
}

func (m *MemoryCache) Del(item interface{}) error {
	key, err := getKey(item)
	if err != nil {
		return err
	}

	keys, err := m.factoryRelevantKeys(key)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, k := range keys {
		if _, ok := m.data[k]; ok {
			delete(m.data, k)
		}
	}
	return nil
}

// Resolve and factory of relevant cahce keys.
// To resolve relevant cahe keys, we access to redis eatch time.
// It might be affect to performance, so we recommend to nesting cahe at least less than 4 or 5.
func (m *MemoryCache) factoryRelevantKeys(key string) ([]string, error) {
	m.mu.Lock()
	b, ok := m.data[key]
	if !ok {
		m.mu.Unlock()
		return nil, fmt.Errorf("failed to get record for delete. Key is %s", key)
	} else if b.Expired() {
		m.mu.Unlock()
		delete(m.data, key)
		return nil, fmt.Errorf("failed to get record for delete. Key is %s", key)
	}
	m.mu.Unlock()

	relevantKeys := []string{key}
	keys, _ := decodeMeta(b.data)
	if keys == nil {
		return relevantKeys, nil
	}
	relevant := bytes.Split(keys, []byte(keyDelimiter))
	for _, v := range relevant {
		rKeys, err := m.factoryRelevantKeys(string(v))
		if err != nil {
			return nil, err
		}
		relevantKeys = append(relevantKeys, rKeys...)
	}
	return relevantKeys, nil
}

var _ Cache = (*MemoryCache)(nil)
