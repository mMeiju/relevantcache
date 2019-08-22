package relevantcache

import (
	"github.com/gomodule/redigo/redis"
)

func (r *RedisCache) Conn() redis.Conn {
	return r.conn
}

func (r *RedisCache) FactoryRelevantKeys(key interface{}) ([]interface{}, error) {
	return r.factoryRelevantKeys(key)
}
