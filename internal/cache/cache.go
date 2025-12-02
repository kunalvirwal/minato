package cache

import (
	"net/http"
	"strconv"
)

const (
	LRUenum = "LRU"
	LFUenum = "LFU" // Future Implementation
)

type Cache interface {

	// To fetch an element from the cache and lazily evict if not found
	Get(key string) (Response, bool)

	// To add an element to the cache
	Set(key string, value Response, expiresAt int64)

	// To get TTL of cache entries
	GetTTL() int64
}

type Response struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

func CreateCache(cacheType string, capacity uint64, maxsize uint64, ttl uint64) Cache {
	if cacheType == LRUenum {
		return &LRUCache{
			Data:     make(map[string]*entry),
			Capacity: int(capacity),
			MaxSize:  int(maxsize),
			Head:     nil,
			Tail:     nil,
			TTL:      int64(ttl),
		}
	}
	// Future Implementation for LFU can be added here
	return nil
}

// Generate key for Cache
// Build cache key takes query parameters into account while building the key,
// so if the request includes time dependent query parameters, then cache might never hit.
func BuildCacheKey(r *http.Request, port int) string {
	key := r.Method + "_" + strconv.Itoa(port) + "_" + r.Host + r.URL.Path
	if r.URL.RawQuery != "" {
		key += "?" + r.URL.RawQuery
	}
	return key
}
