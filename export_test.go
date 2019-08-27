package relevantcache

import (
	"github.com/go-redis/redis"
)

func (r *RedisCache) Conn() *redis.Client {
	return r.conn
}

func (r *RedisCache) FactoryRelevantKeys(key string) []string {
	return r.factoryRelevantKeys(key)
}

func (m *MemoryCache) FactoryRelevantKeys(key string) []string {
	return m.factoryRelevantKeys(key)
}
