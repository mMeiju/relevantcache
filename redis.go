package relevantcache

import (
	"bytes"
	"fmt"
	"strings"

	"crypto/tls"
	"net/url"

	"github.com/gomodule/redigo/redis"
)

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
	key, err := getKey(item)
	if err != nil {
		return nil, err
	}
	b, err := redis.Bytes(r.conn.Do("GET", key))
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
		value = item.encode()
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
	key, err := getKey(item)
	if err != nil {
		return err
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

	relevantKeys := []interface{}{key}
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
