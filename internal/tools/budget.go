package tools

import (
	"container/list"
	"context"
	"sync"
)

// Budget limits how much is being worked on at once, in units of whatever the caller counts (here, bytes of
// message). Work which needs more than is free waits for it, and work is let in in the order it arrived, so
// a large piece of work is not held up for ever by a stream of small ones.
type Budget struct {
	mu      sync.Mutex
	size    int64
	used    int64
	waiters list.List // of *budgetWaiter, the first of which has been waiting the longest
}

type budgetWaiter struct {
	n     int64
	ready chan struct{}
}

// NewBudget returns a budget of size units.
func NewBudget(size int64) *Budget {
	return &Budget{size: size}
}

// Acquire waits until n units are free, takes them, and returns the function which gives them back (which
// must be called once). Work which needs more than the whole budget takes all of it, and so runs on its own.
// Nothing is taken for n of 0 or less. It returns the error of the context, and takes nothing, if the
// context is done before the units are free.
func (b *Budget) Acquire(ctx context.Context, n int64) (func(), error) {
	if n <= 0 {
		return func() {}, nil
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	b.mu.Lock()
	if n > b.size {
		n = b.size
	}

	// nobody is waiting, and there is room
	if b.waiters.Len() == 0 && b.used+n <= b.size {
		b.used += n
		b.mu.Unlock()

		return b.releaseFunc(n), nil
	}

	w := &budgetWaiter{n: n, ready: make(chan struct{})}
	elem := b.waiters.PushBack(w)
	b.mu.Unlock()

	select {
	case <-w.ready:
		return b.releaseFunc(n), nil

	case <-ctx.Done():
		if b.abandon(elem, w) {
			// the units were given just as the context ended, and they are taken
			return b.releaseFunc(n), nil
		}

		return nil, ctx.Err()
	}
}

// abandon takes a waiter out of the queue when its context is done, and returns whether it had already been
// given its units (in which case it has them). If it was the first in line, the ones after it may now fit.
func (b *Budget) abandon(elem *list.Element, w *budgetWaiter) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	select {
	case <-w.ready:
		return true
	default:
	}

	isFirst := b.waiters.Front() == elem
	b.waiters.Remove(elem)
	if isFirst {
		b.grant()
	}

	return false
}

func (b *Budget) releaseFunc(n int64) func() {
	var once sync.Once

	return func() {
		once.Do(func() {
			b.mu.Lock()
			b.used -= n
			b.grant()
			b.mu.Unlock()
		})
	}
}

// grant lets in the ones at the front of the queue for as long as they fit. The lock must be held.
func (b *Budget) grant() {
	for {
		front := b.waiters.Front()
		if front == nil {
			return
		}

		w := front.Value.(*budgetWaiter)
		if b.used+w.n > b.size {
			// the first in line is let in before the ones behind it, even if they would fit
			return
		}

		b.used += w.n
		b.waiters.Remove(front)
		close(w.ready)
	}
}
