# relevantcache

`relevantcache` manages caches with `relevant` to other cache keys.

## Installtion

```
$ go get -u github.com/ysugimoto/relevantcache
```

## Usage

### Simple KVS

Typical usage of simple KVS store:

```Go
import (
    "log"

    rc "github.com/ysugimoto/relevantcache"
)

func main() {
    c, err := rc.NewRedis("redis://127.0.0.1:6379")
    if err != nil {
        log.Fatalln(err)
    }

    if err := c.Set("lorem", "ipsum"); err != nil {
        log.Fatalln(err)
    }
    v, err := c.Get("lorem")
    if err != nil {
        log.Fatalln(err)
    }
    log.Println(string(v)) // -> "ipsum"
}
```

Note that on simple usage, we always return value as `[]byte`.


### Relevant KVS

Use `rc.Item` instead of primitive value:

```Go
import (
    "log"

    rc "github.com/ysugimoto/relevantcache"
)

func main() {
    c, err := rc.NewRedis("redis://127.0.0.1:6379")
    if err != nil {
        log.Fatalln(err)
    }

    // some existing cache
    if err := c.Set("parent01", "foo"); err != nil {
        log.Fatalln(err)
    }
    // use relevant cache
    item := rc.NewItem("child01").RelevantTo("parent01").Value("bar")
    if err := c.Set(item); err != nil {
        log.Fatalln(err)
    }

    // And delete it
    err := c.Del("child01")
    if err != nil {
        log.Fatalln(err)
    }

    // "parent01" also deleted.
}
```

## Describe how it works

This package uses a first of N bytes in cache record as relevant cache metadata, the record format is:

```
[key]: ![relevant cache keys as comma contanetion data]@[split keys (currently no use)]@[cache data...]
```

The first character of `!` indicats (record has a metadata), and following, relevant cache keys, and split keys can be seen with delimiter `@`.

### GET

We retrieve a cache from backend and strip metadata, and return original record. You don't need to care about metadata.

### SET

We make a metadata section and prepend to cache record with some signature which distinguish that is metadata.

### DEL

We retrieve a cache from backend and parse metadata, and delete them as cache key recursibely.


## TLS Connection

When you connect to redis with secure TLS, you can specify endpoint URL that starts with `tls://`:

```Go
c, err := rc.NewRedis("tls://127.0.0.1:6380")
```


## Features

- [x] Redis Backend
- [ ] Memcached Backend

## License

MIT

## Author

Yoshiaki Sugimoto


