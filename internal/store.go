package internal

import (
	"sync"
	"time"

	"github.com/atotto/clipboard"
)

type Store interface {
	Get() (string, time.Time)
	Update(value string)
	Set(value string, updatedAt time.Time)
}

type BasicStore struct {
	mu        *sync.Mutex
	value     string
	updatedAt time.Time
}

func NewBasicStore() (BasicStore, error) {
	clip, err := clipboard.ReadAll()
	if err != nil {
		return BasicStore{}, err
	}
	return BasicStore{
		mu:        new(sync.Mutex),
		value:     clip,
		updatedAt: time.Now(),
	}, nil
}

var (
	_ Store = (*BasicStore)(nil)
)

func (c *BasicStore) Get() (string, time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.value, c.updatedAt
}

func (c *BasicStore) Update(value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value = value
	c.updatedAt = time.Now()
}

func (c *BasicStore) Set(value string, updatedAt time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value = value
	c.updatedAt = updatedAt
}
