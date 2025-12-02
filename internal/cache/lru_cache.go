package cache

import (
	"fmt"
	"sync"
	"time"
)

type LRUCache struct {
	Data     map[string]*entry
	Capacity int
	MaxSize  int
	Head     *entry
	Tail     *entry
	TTL      int64
	Mu       sync.Mutex
}

type entry struct {
	key       string
	value     Response
	prev      *entry // MRU
	next      *entry // LRU
	expiresAt int64
}

func (c *LRUCache) Get(key string) (Response, bool) {
	c.Mu.Lock()
	defer c.Mu.Unlock()

	ent, found := c.Data[key]
	if !found {

		fmt.Println("Cache miss:", key)
		return Response{}, false
	}
	// fmt.Println("Cache hit:", key, string(ent.value))
	now := time.Now().Unix()
	if ent.expiresAt < now {
		// Cache invalidated
		c.detach(ent)
		delete(c.Data, key)
		return Response{}, false
	}

	// Move to front
	c.moveToFront(ent)
	return ent.value, true

}

// Set adds or updates an entry in the cache with provided TTL expiry.
func (c *LRUCache) Set(key string, value Response, expiresAt int64) {
	if c.MaxSize > 0 && len(value.Body) > c.MaxSize {
		// Do not cache if size exceeds MaxSize
		return
	}

	c.Mu.Lock()
	defer c.Mu.Unlock()

	if ent, found := c.Data[key]; found {
		// Update value and move to front
		ent.value = value
		c.moveToFront(ent)
		return
	}

	// Create new entry
	newEnt := &entry{
		key:       key,
		value:     value,
		expiresAt: expiresAt,
		prev:      nil,
		next:      c.Head,
	}
	if c.Head != nil {
		c.Head.prev = newEnt
	}
	c.Head = newEnt
	if c.Tail == nil {
		c.Tail = newEnt
	}
	c.Data[key] = newEnt

	// Check Capacity
	if len(c.Data) > c.Capacity {
		// Remove LRU entry
		oldTail := c.Tail
		c.detach(oldTail)
		delete(c.Data, oldTail.key)
	}
}

// Returns the default TTL for cache entries
func (c *LRUCache) GetTTL() int64 {
	return c.TTL
}

func (c *LRUCache) moveToFront(ent *entry) {

	// If already at front, do nothing
	if ent == c.Head {
		return
	}
	// Detach from current position
	c.detach(ent)

	// Insert at front
	ent.next = c.Head
	if c.Head != nil {
		c.Head.prev = ent
	}
	c.Head = ent
	if c.Tail == nil {
		c.Tail = ent
	}
}

func (c *LRUCache) detach(ent *entry) {
	// Remove from current position
	if ent.prev != nil {
		ent.prev.next = ent.next
	} else {
		c.Head = ent.next
	}
	if ent.next != nil {
		ent.next.prev = ent.prev
	} else {
		c.Tail = ent.prev
	}

	ent.prev = nil
	ent.next = nil
}
