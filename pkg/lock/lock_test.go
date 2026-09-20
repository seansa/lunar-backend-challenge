package lock

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestKeyedMutexSerialisesTheSameKey(t *testing.T) {
	const goroutines = 8

	keyed := NewKeyedMutex()

	var (
		wg        sync.WaitGroup
		inside    int32
		maxInside int32
		counter   int
	)

	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()

			release := keyed.Lock("193270a9-c9cf-404a-8f83-838e71d9ae67")
			defer release()

			current := atomic.AddInt32(&inside, 1)
			for {
				highest := atomic.LoadInt32(&maxInside)
				if current <= highest || atomic.CompareAndSwapInt32(&maxInside, highest, current) {
					break
				}
			}

			counter++
			atomic.AddInt32(&inside, -1)
		}()
	}

	wg.Wait()

	if maxInside != 1 {
		t.Fatalf("the same key must be held by one goroutine at a time, saw %d", maxInside)
	}
	if counter != goroutines {
		t.Fatalf("expected %d critical sections, got %d", goroutines, counter)
	}
	if len(keyed.locks) != 0 {
		t.Fatalf("lock entries must be dropped once released, got %d left", len(keyed.locks))
	}
}

func TestKeyedMutexDoesNotBlockOtherKeys(t *testing.T) {
	keyed := NewKeyedMutex()

	release := keyed.Lock("channel-a")
	defer release()

	acquired := make(chan struct{})
	go func() {
		releaseOther := keyed.Lock("channel-b")
		close(acquired)
		releaseOther()
	}()

	select {
	case <-acquired:
	case <-time.After(time.Second):
		t.Fatal("locking one key must not block a different key")
	}
}
