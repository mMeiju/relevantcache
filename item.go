package relevantcache

import (
	"fmt"
	"strings"
)

// Relevant item struct
// If you have relevant keys, create pointer of this struct and pass to methods
type Item struct {
	key      string
	relevant []*Item
	ttl      int64
	value    interface{}
}

func KeyGen(args ...interface{}) string {
	keys := make([]string, len(args))
	for i, v := range args {
		keys[i] = fmt.Sprint(v)
	}
	return strings.Join(keys, "_")
}

// Crate Item pointer. argment accepts any amounts, types. Arguments becomes cache key.
func NewItem(args ...interface{}) *Item {
	return &Item{
		key:      KeyGen(args...),
		relevant: []*Item{},
	}
}

// Declare relevant cache keys
func (i *Item) RelevantTo(args ...interface{}) *Item {
	i.relevant = append(i.relevant, NewItem(args...))
	return i
}

// Declare relevant cache keys with asterisk sign suffix
func (i *Item) RelevantAll(args ...interface{}) *Item {
	args = append(args, "*")
	i.relevant = append(i.relevant, NewItem(args...))
	return i
}

// Set TTL
func (i *Item) Ttl(ttl int64) *Item {
	i.ttl = ttl
	return i
}

// Set Value
func (i *Item) Value(val interface{}) *Item {
	i.value = val
	return i
}

// Get cache key
func (i *Item) cacheKey() string {
	return i.key
}

// List relevant cache keys
func (i *Item) getRelevaneKeys() []string {
	relevantKeys := make([]string, len(i.relevant))
	for j, v := range i.relevant {
		relevantKeys[j] = v.cacheKey()
	}
	return relevantKeys
}

// Generate and get metadata
func (i *Item) encode() []byte {
	return encodeMeta(
		strings.Join(i.getRelevaneKeys(), keyDelimiter),
		i.value,
	)
}

// Codec: encode metadata and actual data to byte slice for storing
func encodeMeta(keyStr string, value interface{}) []byte {
	keys := []byte(keyStr)
	m := []byte{signatureSign, nb}
	size := len(keys)
	m = append(m, byte(size<<8), byte(size&0xFF))
	m = append(m, keys...)
	m = append(m, []byte(fmt.Sprint(value))...)
	return m
}

// Codec: decode from stored data to metadata and actual data
func decodeMeta(dat []byte) ([]byte, []byte) {
	if len(dat) <= 2 {
		return nil, dat
	} else if dat[0] != signatureSign || dat[1] != nb {
		return nil, dat
	}
	size := int16((dat[2] << 8) | dat[3])
	return dat[4 : 4+size], dat[4+size:]
}

// Consider type and return as type conversion-ed value
func getKey(v interface{}) (string, error) {
	switch t := v.(type) {
	case *Item:
		return t.cacheKey(), nil
	case string:
		return t, nil
	case []byte:
		return string(t), nil
	default:
		return "", fmt.Errorf("Invalid key type. key accepts only string, []byte, and *Item.")
	}
}
