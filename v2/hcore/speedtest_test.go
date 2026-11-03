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
