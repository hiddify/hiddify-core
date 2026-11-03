package hcore

import (
	"bytes"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestUploadBodySendsTheSizeFromOneBlock(t *testing.T) {
	block := uploadBlock()
	for _, size := range []int64{0, 1, int64(len(block)) - 1, int64(len(block)), 3*int64(len(block)) + 17} {
		sent, err := io.ReadAll(&uploadBody{left: size})
		if err != nil {
			t.Fatal(err)
		}
		if int64(len(sent)) != size {
			t.Fatalf("size %d: sent %d", size, len(sent))
		}
		for at := 0; at < len(sent); at += len(block) {
			end := min(at+len(block), len(sent))
			if !bytes.Equal(sent[at:end], block[:end-at]) {
				t.Fatalf("size %d: bytes at %d are not the block's", size, at)
			}
		}
	}
}

func TestSpeedTestsOfAProxyTakeTurns(t *testing.T) {
	var running, most atomic.Int32
	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			release := lockProxyTest("a")
			now := running.Add(1)
			for {
				seen := most.Load()
				if now <= seen || most.CompareAndSwap(seen, now) {
					break
				}
			}
			time.Sleep(10 * time.Millisecond)
			running.Add(-1)
			release()
		}()
	}
	wg.Wait()
	if most.Load() != 1 {
		t.Fatalf("%d tests of one proxy ran at once", most.Load())
	}
	// another proxy's test does not wait for this one's
	release := lockProxyTest("a")
	done := make(chan struct{})
	go func() {
		lockProxyTest("b")()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("a test of b waited for a test of a")
	}
	release()
}

func TestSpeedResultsOutliveTheRunUntilCleared(t *testing.T) {
	saved := sWorkingPath
	sWorkingPath = t.TempDir()
	reset := func() {
		speedTests.Lock()
		speedTests.results, speedTests.loaded = [2]speedResults{}, false
		speedTests.Unlock()
	}
	reset()
	t.Cleanup(func() {
		sWorkingPath = saved
		reset()
	})

	speedTests.Lock()
	setSpeedLocked(upload, "a", false, 1000)
	setSpeedLocked(download, "a", false, -1)
	speedTests.Unlock()
	if speed, _, cached := speedInfo(upload, "a"); speed != 1000 || cached {
		t.Fatalf("this run's speed = %d, cached %v", speed, cached)
	}

	// the core stops: the speeds stay, as from an earlier run
	cacheSpeedTests()
	if speed, _, cached := speedInfo(upload, "a"); speed != 1000 || !cached {
		t.Fatalf("after the stop: %d, cached %v", speed, cached)
	}

	// the app starts again: they are read back, as from an earlier run
	reset()
	if speed, _, cached := speedInfo(download, "a"); speed != -1 || !cached {
		t.Fatalf("after a restart: %d, cached %v", speed, cached)
	}

	// a new test makes it this run's again
	speedTests.Lock()
	setSpeedLocked(upload, "a", false, 2000)
	speedTests.Unlock()
	if speed, _, cached := speedInfo(upload, "a"); speed != 2000 || cached {
		t.Fatalf("after a new test: %d, cached %v", speed, cached)
	}

	clearSpeedTests()
	if speed, _, _ := speedInfo(upload, "a"); speed != 0 {
		t.Fatalf("after clearing: %d", speed)
	}
	reset()
	if speed, _, _ := speedInfo(download, "a"); speed != 0 {
		t.Fatalf("a cleared speed came back after a restart: %d", speed)
	}
}
