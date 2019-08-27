package relevantcache

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
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
	return time.Now().After(m.expiration)
}

type MemoryCache struct {
	data map[string]memoryCacheEntry
	mu   sync.Mutex
	w    io.Writer
}

func NewMemoryCache(opts ...option) *MemoryCache {
	m := &MemoryCache{
		data: make(map[string]memoryCacheEntry),
	}
	for _, o := range opts {
		switch o.name {
		case optionNameDebugWriter:
			m.w = o.value.(io.Writer)
		}
	}
	return m
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
		return nil, fmt.Errorf("record has been expired for key: %s", key)
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
		debug(m.w, fmt.Sprintf("[SET] cahce key %s is relevant to %q\n", key, item.getRelevaneKeys()))
	case 2:
		key = args[0].(string)
		value = args[1]
		ttl = 0
	case 3:
		key = args[0].(string)
		value = args[1]
		ttl = args[2].(int)
	}

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

	m.mu.Lock()
	defer m.mu.Unlock()

	m.data[key] = memoryCacheEntry{
		data:       dat,
		expiration: expiration,
	}
	return nil
}

func (m *MemoryCache) Del(items ...interface{}) error {
	deleteKeys := []string{}

	for _, v := range items {
		key, err := getKey(v)
		if err != nil {
			continue
		}

		keys, err := m.factoryRelevantKeys(key)
		if err != nil {
			continue
		}

		deleteKeys = append(deleteKeys, keys...)
	}

	if len(deleteKeys) == 0 {
		debug(m.w, "[DEL] delete relevant caches are empty. skipped")
		return nil
	}

	debug(m.w, fmt.Sprintf("[DEL] delete relevant caches %q\n", deleteKeys))

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, k := range deleteKeys {
		if _, ok := m.data[k]; ok {
			delete(m.data, k)
		}
	}
	return nil
}

func (m *MemoryCache) Dump() string {
	return fmt.Sprintf("%+v", m.data)
}

// Resolve and factory of relevant cahce keys.
// To resolve relevant cahe keys, we access to redis eatch time.
// It might be affect to performance, so we recommend to nesting cahe at least less than 4 or 5.
func (m *MemoryCache) factoryRelevantKeys(key string) ([]string, error) {
	// When key contains asterisk sign, whe should list as KEYS command to match against keys
	if strings.Contains(key, "*") {
		return m.factoryRelevantKeysWithAsterisk(key)
	}

	record := func(k string) []byte {
		m.mu.Lock()
		defer m.mu.Unlock()

		b, ok := m.data[k]
		if !ok {
			return nil
		}
		if b.Expired() {
			delete(m.data, k)
			return nil
		}
		return b.data
	}(key)

	if record == nil {
		return []string{}, nil
	}

	relevantKeys := []string{key}
	keys, _ := decodeMeta(record)
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

	debug(m.w, fmt.Sprintf("[REL] %s is relevant to %q\n", key, relevantKeys))
	return relevantKeys, nil
}

// Dealing asterisk sign
func (m *MemoryCache) factoryRelevantKeysWithAsterisk(key string) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	regex, err := regexp.Compile(
		strings.ReplaceAll(key, "*", ".*"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to compile regex on dealing asterisk sign: %s", err.Error())
	}
	relevantKeys := []string{}
	for k, v := range m.data {
		if !regex.MatchString(k) {
			continue
		}
		if v.Expired() {
			delete(m.data, k)
			continue
		}
		relevantKeys = append(relevantKeys, k)
	}
	debug(m.w, fmt.Sprintf("[REL-ASTERISK] %s is relevant to %q\n", key, relevantKeys))

	return relevantKeys, nil
}

var _ Cache = (*MemoryCache)(nil)
