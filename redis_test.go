package relevantcache_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	rc "github.com/ysugimoto/relevantcache"
)

const redisUrl = "redis://127.0.0.1:6379"

func TestRedisCacheConnectRedis(t *testing.T) {
	c, err := rc.NewRedisCache(redisUrl)
	assert.NoError(t, err)
	assert.NotNil(t, c)
	c.Close()
}

func TestRedisCacheGetCacheWithSimpleString(t *testing.T) {
	c, _ := rc.NewRedisCache(redisUrl)
	defer c.Close()

	err := c.Conn().Set("foo", "bar", 0).Err()
	assert.NoError(t, err)
	v, err := c.Get("foo")
	assert.NoError(t, err)
	assert.Equal(t, []byte("bar"), v)
}

func TestRedisCacheSetCacheWithPrimitiveData(t *testing.T) {
	c, _ := rc.NewRedisCache(redisUrl)
	defer c.Close()

	err := c.Set("key", "value")
	assert.NoError(t, err)
}

func TestRedisCacheSetCacheWithPrimitiveDataIncludeTTL(t *testing.T) {
	c, _ := rc.NewRedisCache(redisUrl)
	defer c.Close()

	err := c.Set("key", "value", 100)
	assert.NoError(t, err)
}

func TestRedisCacheSetCacheWithItem(t *testing.T) {
	c, _ := rc.NewRedisCache(redisUrl)
	defer c.Close()

	item := rc.NewItem("child", 1).Value("value").Ttl(10)
	err := c.Set(item)
	assert.NoError(t, err)
	v, err := c.Get("child_1")
	assert.NoError(t, err)
	assert.Equal(t, []byte("value"), v)
}

func TestRedisCacheDelCacheWithPrimitiveString(t *testing.T) {
	c, _ := rc.NewRedisCache(redisUrl)
	defer c.Close()

	err := c.Conn().Set("lorem", "ipsum", 0).Err()
	assert.NoError(t, err)
	err = c.Del("lorem")
	assert.NoError(t, err)

	err = c.Conn().Get("lorem").Err()
	assert.Error(t, err)
}

func TestRedisCacheUnlinkCacheWithPrimitiveString(t *testing.T) {
	c, _ := rc.NewRedisCache(redisUrl)
	defer c.Close()

	err := c.Conn().Set("lorem", "ipsum", 0).Err()
	assert.NoError(t, err)
	err = c.Unlink("lorem")
	assert.NoError(t, err)

	err = c.Conn().Get("lorem").Err()
	assert.Error(t, err)
}

func TestRedisCacheFactoryRelevantKeys(t *testing.T) {
	c, _ := rc.NewRedisCache(redisUrl)
	defer c.Close()

	err := c.Conn().Set("parent_1", "parent", 0).Err()
	assert.NoError(t, err)

	// Store with relevant key
	item := rc.NewItem("child", 10).Value("child").RelevantTo("parent", 1)
	assert.NoError(t, c.Set(item))

	keys := c.FactoryRelevantKeys("child_10")
	assert.Len(t, keys, 2)
	assert.Equal(t, keys[0], "child_10")
	assert.Equal(t, keys[1], "parent_1")
}

func TestRedisCacheDelCacheWithRelevantItemRecursively(t *testing.T) {
	c, _ := rc.NewRedisCache(redisUrl)
	defer c.Close()

	err := c.Conn().Set("parent_1", "parent", 0).Err()
	assert.NoError(t, err)

	// Store with relevant key
	item := rc.NewItem("child", 10).Value("child").RelevantTo("parent", 1)
	assert.NoError(t, c.Set(item))
	item = rc.NewItem("ancestor", 100).Value("ancestor").RelevantTo("child", 10)
	assert.NoError(t, c.Set(item))

	// Delete and ensure deleted relevant key
	assert.NoError(t, c.Del("ancestor_100"))
	err = c.Conn().Get("parent_1").Err()
	assert.Error(t, err)
	err = c.Conn().Get("child_10").Err()
	assert.Error(t, err)
}

func TestRedisCacheDelCacheWithRelevantItemRecursivelyByKeyWithAsterisk(t *testing.T) {
	c, _ := rc.NewRedisCache(redisUrl)
	defer c.Close()

	err := c.Conn().Set("parent_1", "parent", 0).Err()
	assert.NoError(t, err)

	// Store with relevant key
	item := rc.NewItem("child", 10).Value("child").RelevantTo("parent", 1)
	assert.NoError(t, c.Set(item))
	item = rc.NewItem("ancestor", 100).Value("ancestor").RelevantTo("child", 10)
	assert.NoError(t, c.Set(item))

	// Delete key with asterisk and ensure deleted relevant keys
	assert.NoError(t, c.Del("ancestor*"))
	err = c.Conn().Get("parent_1").Err()
	assert.Error(t, err)
	err = c.Conn().Get("child_10").Err()
	assert.Error(t, err)
}

func TestRedisCacheUnlinkCacheWithRelevantItemRecursively(t *testing.T) {
	c, _ := rc.NewRedisCache(redisUrl)
	defer c.Close()

	err := c.Conn().Set("parent_1", "parent", 0).Err()
	assert.NoError(t, err)

	// Store with relevant key
	item := rc.NewItem("child", 10).Value("child").RelevantTo("parent", 1)
	assert.NoError(t, c.Set(item))
	item = rc.NewItem("ancestor", 100).Value("ancestor").RelevantTo("child", 10)
	assert.NoError(t, c.Set(item))

	// Delete and ensure deleted relevant key
	assert.NoError(t, c.Unlink("ancestor_100"))
	err = c.Conn().Get("parent_1").Err()
	assert.Error(t, err)
	err = c.Conn().Get("child_10").Err()
	assert.Error(t, err)
}

func TestRedisCacheFactoryRelevantKeysWithAsterisk(t *testing.T) {
	c, _ := rc.NewRedisCache(redisUrl)
	defer c.Close()

	err := c.Set("asterisk_1", "asterisk", 0)
	assert.NoError(t, err)
	err = c.Set("asterisk_2", "asterisk", 0)
	assert.NoError(t, err)

	// Store with relevant key
	item := rc.NewItem("child", 10).Value("child").RelevantTo("asterisk*")
	assert.NoError(t, c.Set(item))

	keys := c.FactoryRelevantKeys("child_10")
	assert.Len(t, keys, 3)
	assert.Contains(t, keys, "child_10")
	assert.Contains(t, keys, "asterisk_1")
	assert.Contains(t, keys, "asterisk_2")
}

func TestRedisCacheIncrement(t *testing.T) {
	c, _ := rc.NewRedisCache(redisUrl)
	defer c.Close()

	err := c.Increment("incr")
	assert.NoError(t, err)
}

func TestRedisCacheMGet(t *testing.T) {
	c, _ := rc.NewRedisCache(redisUrl)
	defer c.Close()

	assert.NoError(t, c.Set("key1", "val1"))
	assert.NoError(t, c.Set("key3", "val3"))
	defer func() {
		c.Del("key1", "key3")
	}()
	values, err := c.MGet("key1", "key2", "key3")
	assert.NoError(t, err)
	assert.Len(t, values, 3)
	assert.Equal(t, []byte("val1"), values[0])
	assert.Nil(t, values[1])
	assert.Equal(t, []byte("val3"), values[2])
}

func TestRedisCacheHSetAndHLen(t *testing.T) {
	c, _ := rc.NewRedisCache(redisUrl)
	defer c.Close()

	assert.NoError(t, c.HSet("hset_key1", "user01", 1))
	defer func() {
		c.Del("hset_key1")
	}()
	v, err := c.HLen("hset_key1")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), v)
}

func TestRedisCacheHSetAndHLenWithRelevantKeys(t *testing.T) {
	c, _ := rc.NewRedisCache(redisUrl)
	defer c.Close()

	item := rc.NewItem("hset_child", 10).Value("child").RelevantTo("asterisk*")
	assert.NoError(t, c.HSet(item, "user01", 1))
	defer func() {
		c.Del(item)
	}()
	v, err := c.HLen(item)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), v)
}

func TestRedisCacheHGet(t *testing.T) {
	c, _ := rc.NewRedisCache(redisUrl)
	defer c.Close()

	assert.NoError(t, c.HSet("hget_key01", "user01", "foobar"))
	defer func() {
		c.Del("hget_key01")
	}()
	v, err := c.HGet("hget_key01", "user01")
	assert.NoError(t, err)
	assert.Equal(t, []byte("foobar"), v)
}

func TestRedisCacheHGetWithRelevantKey(t *testing.T) {
	c, _ := rc.NewRedisCache(redisUrl)
	defer c.Close()

	item := rc.NewItem("hget_child", 10).Value("child").RelevantTo("asterisk*")
	assert.NoError(t, c.HSet(item, "user01", "foobar"))
	defer func() {
		c.Del(item)
	}()
	v, err := c.HGet(item, "user01")
	assert.NoError(t, err)
	assert.Equal(t, []byte("foobar"), v)
}
