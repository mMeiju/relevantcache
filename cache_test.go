package relevantcache_test

import (
	"testing"

	"github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/assert"
	rc "github.com/ysugimoto/relevantcache"
)

const redisUrl = "redis://127.0.0.1:6379"

func TestConnectRedis(t *testing.T) {
	c, err := rc.NewRedis(redisUrl)
	assert.NoError(t, err)
	assert.NotNil(t, c)
	c.Close()
}

func TestGetCacheWithSimpleString(t *testing.T) {
	c, _ := rc.NewRedis(redisUrl)
	defer c.Close()

	_, err := c.Conn().Do("SET", "foo", "bar")
	assert.NoError(t, err)
	v, err := c.Get("foo")
	assert.NoError(t, err)
	assert.Equal(t, []byte("bar"), v)
}

func TestSetCacheWithPrimitiveData(t *testing.T) {
	c, _ := rc.NewRedis(redisUrl)
	defer c.Close()

	err := c.Set("key", "value")
	assert.NoError(t, err)
}

func TestSetCacheWithPrimitiveDataIncludeTTL(t *testing.T) {
	c, _ := rc.NewRedis(redisUrl)
	defer c.Close()

	err := c.Set("key", "value", 100)
	assert.NoError(t, err)
}

func TestSetCacheWithItem(t *testing.T) {
	c, _ := rc.NewRedis(redisUrl)
	defer c.Close()

	item := rc.NewItem("child", 1).Value("value")
	err := c.Set(item)
	assert.NoError(t, err)
	v, err := c.Get("child1")
	assert.NoError(t, err)
	assert.Equal(t, []byte("value"), v)
}

func TestDelCacheWithPrimitiveString(t *testing.T) {
	c, _ := rc.NewRedis(redisUrl)
	defer c.Close()

	_, err := c.Conn().Do("SET", "lorem", "ipsum")
	assert.NoError(t, err)
	err = c.Del("lorem")
	assert.NoError(t, err)

	_, err = redis.String(c.Conn().Do("GET", "lorem"))
	assert.Error(t, err)
}

func TestFactoryRelevantKeys(t *testing.T) {
	c, _ := rc.NewRedis(redisUrl)
	defer c.Close()

	_, err := c.Conn().Do("SET", "parent1", "parent")
	assert.NoError(t, err)

	// Store with relevant key
	item := rc.NewItem("child", 10).Value("child").RelevantTo("parent", 1)
	assert.NoError(t, c.Set(item))

	keys, err := c.FactoryRelevantKeys("child10")
	assert.NoError(t, err)
	assert.Len(t, keys, 2)
	assert.Equal(t, keys[0].(string), "child10")
	assert.Equal(t, keys[1].(string), "parent1")
}

func TestDelCacheWithRelevantItemRecursively(t *testing.T) {
	c, _ := rc.NewRedis(redisUrl)
	defer c.Close()

	_, err := c.Conn().Do("SET", "parent1", "parent")
	assert.NoError(t, err)

	// Store with relevant key
	item := rc.NewItem("child", 10).Value("child").RelevantTo("parent", 1)
	assert.NoError(t, c.Set(item))
	item = rc.NewItem("ancestor", 100).Value("ancestor").RelevantTo("child", 10)
	assert.NoError(t, c.Set(item))

	// Delete and ensure deleted relevant key
	assert.NoError(t, c.Del("ancestor100"))
	_, err = redis.String(c.Conn().Do("GET", "parent1"))
	assert.Error(t, err)
	_, err = redis.String(c.Conn().Do("GET", "child10"))
	assert.Error(t, err)
}
