package relevantcache_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	rc "github.com/ysugimoto/relevantcache"
)

func TestMemoryCacheGetCacheWithSimpleString(t *testing.T) {
	c := rc.NewMemoryCahe()
	defer c.Close()

	err := c.Set("foo", "bar", 0)
	assert.NoError(t, err)
	v, err := c.Get("foo")
	assert.NoError(t, err)
	assert.Equal(t, []byte("bar"), v)
}

func TestMemoryCacheSetCacheWithPrimitiveData(t *testing.T) {
	c := rc.NewMemoryCahe()
	defer c.Close()

	err := c.Set("key", "value")
	assert.NoError(t, err)
}

func TestMemoryCacheSetCacheWithPrimitiveDataIncludeTTL(t *testing.T) {
	c := rc.NewMemoryCahe()
	defer c.Close()

	err := c.Set("key", "value", 100)
	assert.NoError(t, err)
}

func TestMemoryCacheSetCacheWithItem(t *testing.T) {
	c := rc.NewMemoryCahe()
	defer c.Close()

	item := rc.NewItem("child", 1).Value("value")
	err := c.Set(item)
	assert.NoError(t, err)
	v, err := c.Get("child1")
	assert.NoError(t, err)
	assert.Equal(t, []byte("value"), v)
}

func TestMemoryCacheDelCacheWithPrimitiveString(t *testing.T) {
	c := rc.NewMemoryCahe()
	defer c.Close()

	err := c.Set("lorem", "ipsum", 0)
	assert.NoError(t, err)
	err = c.Del("lorem")
	assert.NoError(t, err)

	_, err = c.Get("lorem")
	assert.Error(t, err)
}

func TestMemoryCacheFactoryRelevantKeys(t *testing.T) {
	c := rc.NewMemoryCahe()
	defer c.Close()

	err := c.Set("parent1", "parent", 0)
	assert.NoError(t, err)

	// Store with relevant key
	item := rc.NewItem("child", 10).Value("child").RelevantTo("parent", 1)
	assert.NoError(t, c.Set(item))

	keys, err := c.FactoryRelevantKeys("child10")
	assert.NoError(t, err)
	assert.Len(t, keys, 2)
	assert.Equal(t, keys[0], "child10")
	assert.Equal(t, keys[1], "parent1")
}

func TestMemoryCacheDelCacheWithRelevantItemRecursively(t *testing.T) {
	c := rc.NewMemoryCahe()
	defer c.Close()

	err := c.Set("parent1", "parent", 0)
	assert.NoError(t, err)

	// Store with relevant key
	item := rc.NewItem("child", 10).Value("child").RelevantTo("parent", 1)
	assert.NoError(t, c.Set(item))
	item = rc.NewItem("ancestor", 100).Value("ancestor").RelevantTo("child", 10)
	assert.NoError(t, c.Set(item))

	// Delete and ensure deleted relevant key
	assert.NoError(t, c.Del("ancestor100"))
	_, err = c.Get("parent1")
	assert.Error(t, err)
	_, err = c.Get("child10")
	assert.Error(t, err)
}
