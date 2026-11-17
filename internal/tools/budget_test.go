package tools

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func (b *Budget) state() (used int64, waiting int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.used, b.waiters.Len()
}

// acquireAsync starts an Acquire, and reports when it has it.
func acquireAsync(ctx context.Context, b *Budget, n int64) <-chan func() {
	got := make(chan func(), 1)
	go func() {
		release, err := b.Acquire(ctx, n)
		if err != nil {
			close(got)
			return
		}
		got <- release
	}()

	return got
}

func waitQueued(t *testing.T, b *Budget, waiting int) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, w := b.state(); w == waiting {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("%d are not waiting", waiting)
}

func mustHave(t *testing.T, got <-chan func(), what string) func() {
	t.Helper()

	select {
	case release, ok := <-got:
		if !ok || release == nil {
			t.Fatalf("%s: not acquired", what)
		}
		return release
	case <-time.After(2 * time.Second):
		t.Fatalf("%s: still waiting", what)
	}

	return nil
}

func mustWait(t *testing.T, got <-chan func(), what string) {
	t.Helper()

	select {
	case <-got:
		t.Fatalf("%s: acquired while it should wait", what)
	case <-time.After(150 * time.Millisecond):
	}
}

func TestBudgetTakesAndGivesBack(t *testing.T) {
	b := NewBudget(10)

	r1, err := b.Acquire(context.Background(), 4)
	if err != nil {
		t.Fatal(err)
	}
	r2, _ := b.Acquire(context.Background(), 6)
	if used, _ := b.state(); used != 10 {
		t.Fatalf("used %d, want 10", used)
	}

	r1()
	r1() // giving back twice does not give back twice
	if used, _ := b.state(); used != 6 {
		t.Fatalf("used %d, want 6", used)
	}
	r2()
	if used, _ := b.state(); used != 0 {
		t.Fatalf("used %d, want 0", used)
	}

	// nothing is taken for nothing
	for _, n := range []int64{0, -5} {
		release, err := b.Acquire(context.Background(), n)
		if err != nil || release == nil {
			t.Fatalf("n=%d: %v", n, err)
		}
		release()
		if used, _ := b.state(); used != 0 {
			t.Fatalf("n=%d: used %d", n, used)
		}
	}
}

func TestBudgetWaitsForRoom(t *testing.T) {
	b := NewBudget(10)
	hold, _ := b.Acquire(context.Background(), 6)

	// exploit: more than is free waits
	got := acquireAsync(context.Background(), b, 6)
	mustWait(t, got, "6 with 6 in use")

	// what would fit still waits behind the one in front of it (see the next test)
	small := acquireAsync(context.Background(), b, 4)
	waitQueued(t, b, 2)
	mustWait(t, small, "4 behind 6")

	hold()
	release := mustHave(t, got, "6 once there is room")
	releaseSmall := mustHave(t, small, "4 after the 6, with room for both")
	if used, _ := b.state(); used != 10 {
		t.Fatalf("used %d, want 10", used)
	}
	release()
	releaseSmall()
}

// The one which has waited longest goes first, even when a later one would fit. Otherwise a stream of
// small ones would keep a large one waiting for ever.
func TestBudgetIsFirstComeFirstServed(t *testing.T) {
	b := NewBudget(10)
	hold, _ := b.Acquire(context.Background(), 5)

	big := acquireAsync(context.Background(), b, 10)
	waitQueued(t, b, 1)

	// these fit alongside what is held, but they are behind the big one
	var smalls []<-chan func()
	for range 5 {
		smalls = append(smalls, acquireAsync(context.Background(), b, 1))
	}
	waitQueued(t, b, 6)
	for i, s := range smalls {
		mustWait(t, s, "a small one behind a large one "+string(rune('0'+i)))
	}
	if used, _ := b.state(); used != 5 {
		t.Fatalf("used %d, want 5", used)
	}

	// the large one runs on its own
	hold()
	releaseBig := mustHave(t, big, "the large one")
	if used, _ := b.state(); used != 10 {
		t.Fatalf("used %d, want 10", used)
	}
	for _, s := range smalls {
		mustWait(t, s, "a small one while the large one runs")
	}

	// then the small ones, all together
	releaseBig()
	for _, s := range smalls {
		mustHave(t, s, "a small one after the large one")()
	}
}

func TestBudgetLargerThanTheBudgetRunsAlone(t *testing.T) {
	b := NewBudget(10)

	// when nothing is happening it runs at once, and takes it all
	release, err := b.Acquire(context.Background(), 1000)
	if err != nil {
		t.Fatal(err)
	}
	if used, _ := b.state(); used != 10 {
		t.Fatalf("used %d, want 10 (all of it)", used)
	}

	// and others wait for it
	small := acquireAsync(context.Background(), b, 1)
	mustWait(t, small, "a small one while a very large one runs")
	release()
	mustHave(t, small, "a small one after")()

	// when something is already running it waits, as the others do
	hold, _ := b.Acquire(context.Background(), 1)
	huge := acquireAsync(context.Background(), b, 1000)
	mustWait(t, huge, "a very large one with 1 in use")
	hold()
	mustHave(t, huge, "a very large one once it is free")()
}

func TestBudgetStopsWaitingWhenTheContextIsDone(t *testing.T) {
	b := NewBudget(10)
	hold, _ := b.Acquire(context.Background(), 6)

	// exploit: the first in line gives up, and the one behind it, which would fit, is let in
	first, cancelFirst := context.WithCancel(context.Background())
	firstResult := make(chan error, 1)
	go func() { _, err := b.Acquire(first, 6); firstResult <- err }()
	waitQueued(t, b, 1)

	second := acquireAsync(context.Background(), b, 4)
	waitQueued(t, b, 2)
	mustWait(t, second, "4 behind 6")

	cancelFirst()
	if err := <-firstResult; !errors.Is(err, context.Canceled) {
		t.Fatalf("error %v, want context.Canceled", err)
	}
	mustHave(t, second, "4, once the one in front has gone")()

	// nothing is held by the one which gave up
	hold()
	if used, waiting := b.state(); used != 0 || waiting != 0 {
		t.Fatalf("used %d, waiting %d", used, waiting)
	}

	// a deadline which is reached
	hold, _ = b.Acquire(context.Background(), 10)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	if release, err := b.Acquire(ctx, 1); !errors.Is(err, context.DeadlineExceeded) || release != nil {
		t.Fatalf("%v, %v", release != nil, err)
	}
	if d := time.Since(start); d < 80*time.Millisecond || d > 2*time.Second {
		t.Errorf("waited %v", d)
	}
	hold()

	// control: a context which is already done takes nothing, even when there is room
	done, cancelDone := context.WithCancel(context.Background())
	cancelDone()
	if release, err := b.Acquire(done, 1); !errors.Is(err, context.Canceled) || release != nil {
		t.Fatalf("%v, %v", release != nil, err)
	}
	if used, waiting := b.state(); used != 0 || waiting != 0 {
		t.Fatalf("used %d, waiting %d", used, waiting)
	}
}

// Many at once, some giving up as others give back: the total is never over the budget, and nothing is
// left taken or waiting at the end.
func TestBudgetUnderLoad(t *testing.T) {
	const size = 10
	b := NewBudget(size)

	var inUse, maxInUse atomic.Int64
	var wg sync.WaitGroup

	for i := range 400 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(i)))

			n := int64(rng.Intn(14) + 1) // some are larger than the budget
			ctx := context.Background()
			if rng.Intn(3) == 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, time.Duration(rng.Intn(3000))*time.Microsecond)
				defer cancel()
			}

			release, err := b.Acquire(ctx, n)
			if err != nil {
				return
			}

			now := inUse.Add(min(n, size))
			for {
				m := maxInUse.Load()
				if now <= m || maxInUse.CompareAndSwap(m, now) {
					break
				}
			}
			time.Sleep(time.Duration(rng.Intn(500)) * time.Microsecond)
			inUse.Add(-min(n, size))
			release()
		}()
	}
	wg.Wait()

	if m := maxInUse.Load(); m > size {
		t.Errorf("%d in use at once, the budget is %d", m, size)
	}
	if used, waiting := b.state(); used != 0 || waiting != 0 {
		t.Errorf("used %d, waiting %d at the end", used, waiting)
	}
}

// When the context ends just as the units are given, the waiter has them (and gives them back), and when
// it ends before, it has nothing and the queue is as it was without it.
func TestBudgetAbandon(t *testing.T) {
	// the units were already given: it keeps them, and nothing is taken or given a second time
	b := NewBudget(10)
	hold, _ := b.Acquire(context.Background(), 10)

	w := &budgetWaiter{n: 4, ready: make(chan struct{})}
	elem := b.waiters.PushBack(w)
	hold() // gives them to the waiter
	if used, waiting := b.state(); used != 4 || waiting != 0 {
		t.Fatalf("after the grant: used %d, waiting %d", used, waiting)
	}
	if !b.abandon(elem, w) {
		t.Fatal("abandon did not report that the units were given")
	}
	if used, waiting := b.state(); used != 4 || waiting != 0 {
		t.Fatalf("abandon changed it: used %d, waiting %d", used, waiting)
	}
	b.releaseFunc(4)()
	if used, _ := b.state(); used != 0 {
		t.Fatalf("used %d at the end", used)
	}

	// not given yet: it is taken out, and the one behind it, which fits, is let in
	b = NewBudget(10)
	hold, _ = b.Acquire(context.Background(), 6)
	first := &budgetWaiter{n: 6, ready: make(chan struct{})}
	firstElem := b.waiters.PushBack(first)
	second := &budgetWaiter{n: 4, ready: make(chan struct{})}
	b.waiters.PushBack(second)

	if b.abandon(firstElem, first) {
		t.Fatal("abandon reported units which were not given")
	}
	select {
	case <-second.ready:
	default:
		t.Fatal("the one behind was not let in")
	}
	if used, waiting := b.state(); used != 10 || waiting != 0 {
		t.Fatalf("used %d, waiting %d", used, waiting)
	}
	hold()
}
