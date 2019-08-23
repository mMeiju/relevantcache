package relevantcache

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"crypto/tls"
	"net/url"

	"github.com/go-redis/redis"
)

// Redis backend struct
type RedisCache struct {
	conn *redis.Client
}

// Create RedisCache pointer with some options
// Currently enabled options are:
//
// rc.WithSkipTLSVerify(bool): Skip TLS verification
func NewRedisCache(endpoint string, opts ...option) (*RedisCache, error) {
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
	}, nil
}

// Close connection
func (r *RedisCache) Close() error {
	return r.conn.Close()
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
func (r *RedisCache) Del(item interface{}) error {
	key, err := getKey(item)
	if err != nil {
		return err
	}

	keys, err := r.factoryRelevantKeys(key)
	if err != nil {
		return err
	}

	return r.conn.Del(keys...).Err()
}

// Resolve and factory of relevant cahce keys.
// To resolve relevant cahe keys, we access to redis eatch time.
// It might be affect to performance, so we recommend to nesting cahe at least less than 4 or 5.
func (r *RedisCache) factoryRelevantKeys(key string) ([]string, error) {
	b, err := r.conn.Get(key).Bytes()
	if err != nil {
		return nil, fmt.Errorf("failed to get record for delete. Key is %v, %s", key, err.Error())
	}

	relevantKeys := []string{key}
	keys, _ := decodeMeta(b)
	if keys == nil {
		return relevantKeys, nil
	}
	relevant := bytes.Split(keys, []byte(keyDelimiter))
	for _, v := range relevant {
		rKeys, err := r.factoryRelevantKeys(string(v))
		if err != nil {
			return nil, err
		}
		relevantKeys = append(relevantKeys, rKeys...)
	}
	return relevantKeys, nil
}

var _ Cache = (*RedisCache)(nil)
