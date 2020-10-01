package relevantcache

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"

	"crypto/tls"
	"net/url"

	"github.com/go-redis/redis"
)

var RedisNil = redis.Nil

// Redis backend struct
type RedisCache struct {
	conn *redis.Client
	w    io.Writer
}

// Create RedisCache pointer with some options
// Currently enabled options are:
//
// rc.WithSkipTLSVerify(bool): Skip TLS verification
func NewRedisCache(endpoint string, opts ...option) (*RedisCache, error) {
	var skipVerify bool
	var w io.Writer
	for _, o := range opts {
		switch o.name {
		case optionNameSkipTLSVerify:
			skipVerify = o.value.(bool)
			// case optionNameSplitBufferSize:
			// 	splitChunkSize = o.value.(int64)
		case optionNameDebugWriter:
			w = o.value.(io.Writer)
		}
	}

	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	options := &redis.Options{
		Addr: u.Host,
	}
	if u.Scheme == tlsProtocol {
		hp := strings.SplitN(u.Host, ":", 2)
		options.TLSConfig = &tls.Config{
			ServerName:         hp[0],
			InsecureSkipVerify: false,
		}
		if skipVerify {
			options.TLSConfig.InsecureSkipVerify = true
		}
	}
	conn := redis.NewClient(options)
	if pong, err := conn.Ping().Result(); err != nil {
		return nil, err
	} else if pong != "PONG" {
		return nil, fmt.Errorf("failed to receive PONG from server")
	}
	return &RedisCache{
		conn: conn,
		w:    w,
	}, nil
}

// Close connection
func (r *RedisCache) Close() error {
	return r.conn.Close()
}

// Purge all caches
func (r *RedisCache) Purge() error {
	return r.conn.FlushDBAsync().Err()
}

func (r *RedisCache) Increment(key string) error {
	return r.conn.Incr(key).Err()
}

// Wrap of redis.GET
// item is acceptable either of string of *Item
func (r *RedisCache) Get(item interface{}) ([]byte, error) {
	key, err := getKey(item)
	if err != nil {
		return nil, err
	}
	b, err := r.conn.Get(key).Bytes()
	if err != nil {
		return nil, err
	}
	_, data := decodeMeta(b)
	return data, nil

}

func (r *RedisCache) Dump() string {
	keys, _ := r.conn.Keys("*").Result()
	return fmt.Sprintf("%q", keys)
}

// Wrap of redis.SET/redis.SETEX
// args is acceptable with following argument counts:
//
// count is 1: deal with *Item
// count is 2: deal with first argument as cache key, second argument as value. TTL is 0 (no expiration)
// count is 3: deal with first argument as cache key, second argument as value, third argument as TTL
func (r *RedisCache) Set(args ...interface{}) (err error) {
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
		debug(r.w, fmt.Sprintf("[SET] cahce key %s is relevant to %q\n", key, item.getRelevaneKeys()))
	case 2:
		key = args[0].(string)
		value = args[1]
		ttl = 0
	case 3:
		key = args[0].(string)
		value = args[1]
		ttl = args[2].(int)
	}

	var expire time.Duration
	if ttl > 0 {
		expire = time.Duration(ttl) * time.Second
	}
	return r.conn.Set(key, value, expire).Err()
}

// Wrap of redis.DEL
// item is acceptable either of string of *Item
func (r *RedisCache) Del(items ...interface{}) error {
	deleteKeys := []string{}

	for _, v := range items {
		key, err := getKey(v)
		if err != nil {
			debug(r.w, fmt.Sprintf("[DEL] invalid keys:%v,  %s\n", v, err.Error()))
			continue
		}
		debug(r.w, fmt.Sprintf("[DEL] key is: %s\n", key))

		keys := r.factoryRelevantKeys(key)
		debug(r.w, fmt.Sprintf("[DEL] factory keys are: %q\n", keys))

		deleteKeys = append(deleteKeys, keys...)
	}

	if len(deleteKeys) == 0 {
		debug(r.w, "[DEL] delete relevant caches are empty. skipped\n")
		return nil
	}

	debug(r.w, fmt.Sprintf("[DEL] delete relevant caches %q\n", deleteKeys))
	return r.conn.Unlink(deleteKeys...).Err()
}

// Resolve and factory of relevant cahce keys.
// To resolve relevant cahe keys, we access to redis eatch time.
// It might be affect to performance, so we recommend to nesting cahe at least less than 4 or 5.
func (r *RedisCache) factoryRelevantKeys(key string) []string {
	// When key contains asterisk sign, whe should list as KEYS command to match against keys
	if strings.Contains(key, "*") {
		return r.factoryRelevantKeysWithAsterisk(key)
	}

	relevantKeys := []string{}
	b, err := r.conn.Get(key).Bytes()
	if err != nil {
		debug(r.w, fmt.Sprintf("failed to get record for delete. Key is %v, %s\n", key, err.Error()))
		return relevantKeys
	}

	relevantKeys = append(relevantKeys, key)
	keys, _ := decodeMeta(b)
	if keys == nil {
		return relevantKeys
	}
	relevant := bytes.Split(keys, []byte(keyDelimiter))
	for _, v := range relevant {
		rKeys := r.factoryRelevantKeys(string(v))
		relevantKeys = append(relevantKeys, rKeys...)
	}
	debug(r.w, fmt.Sprintf("[REL] %s is relevant to %q\n", key, relevantKeys))
	return relevantKeys
}

// Dealing asterisk sign
func (r *RedisCache) factoryRelevantKeysWithAsterisk(key string) []string {
	relevantKeys := []string{}
	keys, err := r.conn.Keys(key).Result()
	if err != nil {
		debug(r.w, fmt.Sprintf("failed to list keys for %s, %s\n", key, err.Error()))
		return relevantKeys
	}
	for _, k := range keys {
		ks := r.factoryRelevantKeys(k)
		relevantKeys = append(relevantKeys, ks...)
	}
	debug(r.w, fmt.Sprintf("[REL-ASTERISK] %s is relevant to %q\n", key, relevantKeys))
	return relevantKeys
}

func (r *RedisCache) MGet(keys ...interface{}) ([][]byte, error) {
	cacheKeys := make([]string, len(keys))
	for i, k := range keys {
		key, err := getKey(k)
		if err != nil {
			return nil, err
		}
		cacheKeys[i] = key
	}
	result, err := r.conn.MGet(cacheKeys...).Result()
	if err != nil {
		return nil, err
	}
	ret := make([][]byte, len(cacheKeys))
	for i, v := range result {
		if v == nil {
			ret[i] = nil
			continue
		}
		str := v.(string)
		_, data := decodeMeta([]byte(str))
		ret[i] = data
	}

	return ret, nil
}

var _ Cache = (*RedisCache)(nil)
