// relevantcache package manages "relevant" cache keys in some cache backend (now redis).
// Or, you can use as simple KVS storage, but cannot use redis specific features
// like MGET, HGET, HMGET, ..., you can use GET/SET/DEL only.
//
// Describe:
//
// occationally some of caches are relevant like:
//  cacheA -> cacheB -> cacheC ...
// Then, when cacheA has been destroied, we also delete cacheB because they are relevant.
// And, cacheB has been destoried, cacheC will be destoried.
//
// Strategy:
//
// Just simple. we prepend metadata that is relevant cache keys when store a cache data.
// And when we get it, retrieve metadata and return cache data only.
// And when we delete it, extract metadata ,find relevant keys, and delete recursively.
package relevantcache

import (
	"bytes"
	"fmt"
	"strings"

	"crypto/tls"
	"net/url"

	"github.com/gomodule/redigo/redis"
)

const (
	// Definetely TLS protocol name
	// e.g. tls://[host]:[port] -> connect with TLS
	// e.g. redis://[host]:[port] -> connect without TLS
	tlsProtocol = "tls"
)

var (
	// default delimiters. Typically we don't need to change
	delimiter    = []byte("@")
	delimiterStr = "@"

	relDelimiter    = []byte(",")
	relDelimiterStr = ","
)

// All methods accepts as interface{} because argument can be passed as string or *Item
type Cache interface {
	Get(item interface{}) ([]byte, error)
	Set(item interface{}) error
	Del(item interface{}) error
	Close() error
}

// Relevant item struct
// If you have relevant keys, create pointer of this struct and pass to methods
type Item struct {
	key      string
	relevant []*Item
	ttl      int64
	value    interface{}
}

// Crate Item pointer. argment accepts any amounts, types. Arguments becomes cache key.
func NewItem(args ...interface{}) *Item {
	return &Item{
		key:      fmt.Sprint(args...),
		relevant: []*Item{},
	}
}

// Declare relevant cache keys
func (i *Item) RelevantTo(args ...interface{}) *Item {
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

// Generate and get metadata
func (i *Item) meta() string {
	relevantKeys := make([]string, len(i.relevant))
	for j, v := range i.relevant {
		relevantKeys[j] = v.cacheKey()
	}
	// Cache format is:
	// ![relevant_keys]@[split_num]@[cache_value]
	return fmt.Sprintf(
		"!%s%s%s%s",
		strings.Join(relevantKeys, relDelimiterStr),
		delimiterStr,
		"",
		delimiterStr,
	)
}

// Redis backend struct
type RedisCache struct {
	conn redis.Conn
}

// Create RedisCache pointer with some options
// Currently enabled options are:
//
// rc.WithSkipTLSVerify(bool): Skip TLS verification
func NewRedis(endpoint string, opts ...option) (*RedisCache, error) {
	var skipVerify bool
	for _, o := range opts {
		switch o.name {
		case optionNameSkipTLSVerify:
			skipVerify = o.value.(bool)
			// case optionNameSplitBufferSize:
			// 	splitChunkSize = o.value.(int64)
		}
	}

	u, err := url.Parse(endpoint)
	var options []redis.DialOption
	if u.Scheme == tlsProtocol {
		hp := strings.SplitN(u.Host, ":", 2)
		options = append(
			options,
			redis.DialUseTLS(true),
			redis.DialTLSConfig(&tls.Config{ServerName: hp[0]}),
		)
		if skipVerify {
			options = append(options, redis.DialTLSSkipVerify(true))
		}
	}
	conn, err := redis.Dial("tcp", u.Host, options...)
	if err != nil {
		return nil, err
	}
	return &RedisCache{
		conn: conn,
	}, nil
}

// Close connection
func (r *RedisCache) Close() error {
	return r.conn.Close()
}

// Wrap of redis.GET
// item is acceptable either of string of *Item
func (r *RedisCache) Get(item interface{}) ([]byte, error) {
	var key interface{}
	switch t := item.(type) {
	case *Item:
		key = t.cacheKey()
	case string:
		key = t
	case []byte:
		key = string(t)
	default:
		return nil, fmt.Errorf("Invalid key type. key accepts only string, []byte, and *Item.")
	}

	b, err := redis.Bytes(r.conn.Do("GET", key))
	if err != nil {
		return nil, err
	}
	if b[0] != '!' {
		return b, nil
	}

	splitBytes := bytes.SplitN(b[1:], delimiter, 3)
	if len(splitBytes) != 3 {
		return nil, fmt.Errorf("malformed value format")
	}
	return splitBytes[2], nil
}

// Wrap of redis.SET/redis.SETEX
// args is acceptable with following argument counts:
//
// count is 1: deal with *Item
// count is 2: deal with first argument as cache key, second argument as value. TTL is 0 (no expiration)
// count is 3: deal with first argument as cache key, second argument as value, third argument as TTL
func (r *RedisCache) Set(args ...interface{}) (err error) {
	var key interface{}
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
		value = fmt.Sprintf("%s%s", item.meta(), item.value)
		ttl = int(item.ttl)
	case 2:
		key = args[0]
		value = args[1]
		ttl = 0
	case 3:
		key = args[0]
		value = args[1]
		ttl = args[2].(int)
	}

	if ttl > 0 {
		_, err = r.conn.Do("SETEX", key, ttl, value)
	} else {
		_, err = r.conn.Do("SET", key, value)
	}
	return err
}

// Wrap of redis.DEL
// item is acceptable either of string of *Item
func (r *RedisCache) Del(item interface{}) error {
	var key interface{}
	switch t := item.(type) {
	case *Item:
		key = t.cacheKey()
	case string:
		key = t
	case []byte:
		key = string(t)
	default:
		return fmt.Errorf("Invalid key type. key accepts only string, []byte, and *Item.")
	}

	keys, err := r.factoryRelevantKeys(key)
	if err != nil {
		return err
	}

	_, err = r.conn.Do("DEL", keys...)
	return err
}

// Resolve and factory of relevant cahce keys.
// To resolve relevant cahe keys, we access to redis eatch time.
// It might be affect to performance, so we recommend to nesting cahe at least less than 4 or 5.
func (r *RedisCache) factoryRelevantKeys(key interface{}) ([]interface{}, error) {
	b, err := redis.Bytes(r.conn.Do("GET", key))
	if err != nil {
		return nil, fmt.Errorf("failed to get record for delete. Key is %v, %s", key, err.Error())
	}

	keys := []interface{}{key}
	if b[0] != '!' {
		return keys, nil
	}

	splitBytes := bytes.SplitN(b[1:], delimiter, 3)
	if len(splitBytes) != 3 {
		return nil, fmt.Errorf("malformed value format")
	}
	relevant := bytes.Split(splitBytes[0], relDelimiter)
	for _, v := range relevant {
		rKeys, err := r.factoryRelevantKeys(string(v))
		if err != nil {
			return nil, err
		}
		keys = append(keys, rKeys...)
	}
	return keys, nil
}
