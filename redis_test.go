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

	item := rc.NewItem("child", 1).Value("value")
	err := c.Set(item)
	assert.NoError(t, err)
	v, err := c.Get("child1")
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

func TestRedisCacheFactoryRelevantKeys(t *testing.T) {
	c, _ := rc.NewRedisCache(redisUrl)
	defer c.Close()

	err := c.Conn().Set("parent_1", "parent", 0).Err()
	assert.NoError(t, err)

	// Store with relevant key
	item := rc.NewItem("child", 10).Value("child").RelevantTo("parent", 1)
	assert.NoError(t, c.Set(item))

	keys, err := c.FactoryRelevantKeys("child_10")
	assert.NoError(t, err)
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

	keys, err := c.FactoryRelevantKeys("child_10")
	assert.NoError(t, err)
	assert.Len(t, keys, 3)
	assert.Contains(t, keys, "child_10")
	assert.Contains(t, keys, "asterisk_1")
	assert.Contains(t, keys, "asterisk_2")
}
