package go_in_memory_cache

import (
	"errors"
	"sync"
	"time"
)

type CacheInterface interface {
	Set(key string, value interface{}, duration time.Duration) error
	Get(key string) (interface{}, bool)
	GetItem(key string) (*Item, bool)
	Delete(key string) error
	Count() int
	Rename(key, newKey string) error
}

type Cache struct {
	sync.RWMutex
	defaultLifetime time.Duration
	cleanupInterval time.Duration
	items           map[string]Item
}

type Item struct {
	Value   interface{}
	Created time.Time
	Expired int64
}

func New(defaultLifetime, cleanupInterval time.Duration) *Cache {
	items := make(map[string]Item)

	cache := Cache{
		defaultLifetime: defaultLifetime,
		cleanupInterval: cleanupInterval,
		items:           items,
	}

	if cleanupInterval > 0 {
		cache.StartGC()
	}

	return &cache
}

func (c *Cache) Set(key string, value interface{}, duration time.Duration) error {
	var expiration int64

	if duration == 0 {
		duration = c.defaultLifetime
	}

	if duration > 0 {
		expiration = time.Now().Add(duration).UnixNano()
	}

	_, ok := c.items[key]
	if ok {
		return errors.New("key already exists")
	}

	c.Lock()
	defer c.Unlock()

	c.items[key] = Item{
		Value:   value,
		Expired: expiration,
		Created: time.Now(),
	}

	return nil
}

func (c *Cache) Get(key string) (interface{}, bool) {
	c.RLock()
	defer c.RUnlock()

	result, ok := c.items[key]
	if !ok {
		return nil, false
	}

	if result.Expired > 0 {
		if time.Now().UnixNano() > result.Expired {
			return nil, false
		}
	}

	return result.Value, true
}

func (c *Cache) GetItem(key string) (*Item, bool) {
	c.RLock()
	defer c.RUnlock()

	result, ok := c.items[key]
	if !ok {
		return nil, false
	}

	if result.Expired > 0 {
		if time.Now().UnixNano() > result.Expired {
			return nil, false
		}
	}

	return &result, true
}

func (c *Cache) Delete(key string) error {
	c.Lock()
	defer c.Unlock()

	if _, ok := c.items[key]; !ok {
		return errors.New("key not found")
	}

	delete(c.items, key)
	return nil
}

func (c *Cache) StartGC() {
	go c.GC()
}

func (c *Cache) GC() {

	for {
		<-time.After(c.cleanupInterval)

		if c.items == nil {
			return
		}

		if keys := c.expiredKeys(); len(keys) > 0 {
			c.ClearItems(keys)
		}

	}
}

func (c *Cache) expiredKeys() (keys []string) {
	c.RLock()
	defer c.RUnlock()

	for key, item := range c.items {
		if time.Now().UnixNano() > item.Expired && item.Expired > 0 {
			keys = append(keys, key)
		}
	}
	return
}

func (c *Cache) ClearItems(keys []string) {
	c.Lock()
	defer c.Unlock()
	for _, key := range keys {
		delete(c.items, key)
	}
}

func (c *Cache) Count() int {
	c.RLock()
	n := len(c.items)
	c.RUnlock()
	return n
}

func (c *Cache) Rename(key string, newKey string) error {
	item, ok := c.GetItem(key)
	if !ok {
		return errors.New("key not found")
	}
	err := c.Delete(key)
	if err != nil {
		return err
	}
	c.Lock()
	defer c.Unlock()
	c.items[newKey] = Item{
		Value:   item.Value,
		Created: item.Created,
		Expired: item.Expired,
	}
	return nil
}

func (c *Cache) Copy(key, newKey string) error {
	item, ok := c.GetItem(key)
	if !ok {
		return errors.New("key not found")
	}

	c.Lock()
	defer c.Unlock()
	c.items[key] = Item{
		Value:   item.Value,
		Created: item.Created,
		Expired: item.Expired,
	}
	return nil
}
