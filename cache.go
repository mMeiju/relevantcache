// relevantcache package manages "relevant" cache keys in some cache backend (now redis).
//
// Describe:
//
// Occasionally some of caches have relevant like:
//  cacheA -> cacheB -> cacheC ...
//
// When cacheA has been destroyed, we also delete cacheB because they are relevant.
// And, when cacheB has been destroyed, cacheC will be destroyed.
//
// Strategy:
//
// Just simple. we prepend metadata that is relevant cache keys when store a cache data.
// When we get it, retrieve metadata and return cache data only,
// and when we delete it, extract metadata, factory relevant keys recursively, and delete those.
package relevantcache

import (
	"io"
)

const (
	// Definetely TLS protocol name
	// e.g. tls://[host]:[port] -> connect with TLS
	// e.g. redis://[host]:[port] -> connect without TLS
	tlsProtocol = "tls"
)

var (
	keyDelimiter  = "|"
	signatureSign = byte('$')
	nb            = byte(0)
)

// All methods accepts as interface{} because argument can be passed as string or *Item
type Cache interface {
	Get(item interface{}) ([]byte, error)
	Set(args ...interface{}) error
	Del(items ...interface{}) error
	Unlink(items ...interface{}) error
	Increment(key string) error
	Close() error
	Dump() string
	Purge() error
	MGet(keys ...interface{}) ([][]byte, error)
	HSet(key interface{}, field string, value interface{}) error
	HLen(key interface{}) (int64, error)
	HGet(key interface{}, field string) ([]byte, error)
}

func debug(w io.Writer, message string) {
	if w != nil {
		io.WriteString(w, message)
	}
}
