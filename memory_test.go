package relevantcache_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	rc "github.com/ysugimoto/relevantcache"
)

func TestMemoryCacheGetCacheWithSimpleString(t *testing.T) {
	c := rc.NewMemoryCache()
	defer c.Close()

	err := c.Set("foo", "bar", 0)
	assert.NoError(t, err)
	v, err := c.Get("foo")
	assert.NoError(t, err)
	assert.Equal(t, []byte("bar"), v)
}

func TestMemoryCacheSetCacheWithPrimitiveData(t *testing.T) {
	c := rc.NewMemoryCache()
	defer c.Close()

	err := c.Set("key", "value")
	assert.NoError(t, err)
}

func TestMemoryCacheSetCacheWithPrimitiveDataIncludeTTL(t *testing.T) {
	c := rc.NewMemoryCache()
	defer c.Close()

	err := c.Set("key", "value", 100)
	assert.NoError(t, err)
}

func TestMemoryCacheSetCacheWithItem(t *testing.T) {
	c := rc.NewMemoryCache()
	defer c.Close()

	item := rc.NewItem("child", 1).Value("value")
	err := c.Set(item)
	assert.NoError(t, err)
	v, err := c.Get("child_1")
	assert.NoError(t, err)
	assert.Equal(t, []byte("value"), v)
}

func TestMemoryCacheDelCacheWithPrimitiveString(t *testing.T) {
	c := rc.NewMemoryCache()
	defer c.Close()

	err := c.Set("lorem", "ipsum", 0)
	assert.NoError(t, err)
	err = c.Del("lorem")
	assert.NoError(t, err)

	_, err = c.Get("lorem")
	assert.Error(t, err)
}

func TestMemoryCacheUnlinkCacheWithPrimitiveString(t *testing.T) {
	c := rc.NewMemoryCache()
	defer c.Close()

	err := c.Set("lorem", "ipsum", 0)
	assert.NoError(t, err)
	err = c.Unlink("lorem")
	assert.NoError(t, err)

	_, err = c.Get("lorem")
	assert.Error(t, err)
}

func TestMemoryCacheFactoryRelevantKeys(t *testing.T) {
	c := rc.NewMemoryCache()
	defer c.Close()

	err := c.Set("parent_1", "parent", 0)
	assert.NoError(t, err)

	// Store with relevant key
	item := rc.NewItem("child", 10).Value("child").RelevantTo("parent", 1)
	assert.NoError(t, c.Set(item))

	keys := c.FactoryRelevantKeys("child_10")
	assert.Len(t, keys, 2)
	assert.Equal(t, keys[0], "child_10")
	assert.Equal(t, keys[1], "parent_1")
}

func TestMemoryCacheDelCacheWithRelevantItemRecursively(t *testing.T) {
	c := rc.NewMemoryCache()
	defer c.Close()

	err := c.Set("parent_1", "parent", 0)
	assert.NoError(t, err)

	// Store with relevant key
	item := rc.NewItem("child", 10).Value("child").RelevantTo("parent", 1)
	assert.NoError(t, c.Set(item))
	item = rc.NewItem("ancestor", 100).Value("ancestor").RelevantTo("child", 10)
	assert.NoError(t, c.Set(item))

	// Delete and ensure deleted relevant key
	assert.NoError(t, c.Del("ancestor_100"))
	_, err = c.Get("parent_1")
	assert.Error(t, err)
	_, err = c.Get("child_10")
	assert.Error(t, err)
}

func TestMemoryCacheUnlinkCacheWithRelevantItemRecursively(t *testing.T) {
	c := rc.NewMemoryCache()
	defer c.Close()

	err := c.Set("parent_1", "parent", 0)
	assert.NoError(t, err)

	// Store with relevant key
	item := rc.NewItem("child", 10).Value("child").RelevantTo("parent", 1)
	assert.NoError(t, c.Set(item))
	item = rc.NewItem("ancestor", 100).Value("ancestor").RelevantTo("child", 10)
	assert.NoError(t, c.Set(item))

	// Delete and ensure deleted relevant key
	assert.NoError(t, c.Unlink("ancestor_100"))
	_, err = c.Get("parent_1")
	assert.Error(t, err)
	_, err = c.Get("child_10")
	assert.Error(t, err)
}

func TestMemoryCacheFactoryRelevantKeysWithAsterisk(t *testing.T) {
	c := rc.NewMemoryCache()
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

func TestMemoryCacheIncrement(t *testing.T) {
	c := rc.NewMemoryCache()
	defer c.Close()

	err := c.Increment("incr")
	assert.NoError(t, err)
}

func TestMemoryCacheMGet(t *testing.T) {
	c := rc.NewMemoryCache()
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
