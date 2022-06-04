package internal

import (
	"context"
	"time"

	"github.com/atotto/clipboard"
)

type Observer interface {
	GetChan() <-chan string
}

type PollingObserver struct {
	ch chan string
}

func NewPollingObserver() PollingObserver {
	return PollingObserver{
		ch: make(chan string, 4),
	}
}

func (c *PollingObserver) poll() error {
	v, err := clipboard.ReadAll()
	if err != nil {
		return err
	}
	c.ch <- v
	return nil
}

func (c *PollingObserver) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(time.Second):
			if err := c.poll(); err != nil {
				return err
			}
		}
	}
}

var (
	_ Observer = (*PollingObserver)(nil)
)

func (c *PollingObserver) GetChan() <-chan string {
	return c.ch
}
