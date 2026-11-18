package hcore

import (
	"bytes"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/adapter/outbound"
	C "github.com/sagernet/sing-box/constant"
	"github.com/stretchr/testify/require"
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

type speedGroup struct {
	fakeProxy
	members []string
}

func (g *speedGroup) All() []string                     { return g.members }
func (g *speedGroup) Selected(string) adapter.Outbound  { return nil }
func (g *speedGroup) AttachConnection(io.Closer) func() { return func() {} }

type fakeSelector struct{ speedGroup }

func (s *fakeSelector) SelectOutbound(string) bool { return true }

// A speed test of lowest or balance measures the proxy the group stands for, which its row shows,
// not every member again; a test of the selector is the test of the list, of its proxies in use.
func TestSpeedTestOfAGroupMeasuresWhatItShows(t *testing.T) {
	p1 := &fakeProxy{outbound.NewAdapter(C.TypeVLESS, "p1", nil, nil)}
	p2 := &fakeProxy{outbound.NewAdapter(C.TypeVLESS, "p2", nil, nil)}
	p3 := &fakeProxy{outbound.NewAdapter(C.TypeVLESS, "p3", nil, nil)}
	lowest := &speedGroup{fakeProxy: fakeProxy{outbound.NewAdapter(C.TypeBalancer, "lowest", nil, nil)}, members: []string{"p1", "p2", "p3"}}
	selector := &fakeSelector{speedGroup{fakeProxy: fakeProxy{outbound.NewAdapter(C.TypeSelector, "select", nil, nil)}, members: []string{"lowest", "p1", "p2", "p3"}}}
	all := map[string]adapter.Outbound{"p1": p1, "p2": p2, "p3": p3, "lowest": lowest, "select": selector}
	lookup := func(tag string) (adapter.Outbound, bool) { out, ok := all[tag]; return out, ok }
	rests := func(tag string) bool { return tag == "p3" }
	member := func(group string) string {
		if group == "lowest" {
			return "p2"
		}
		return group
	}
	tags := func(outs []adapter.Outbound) (tags []string) {
		for _, out := range outs {
			tags = append(tags, out.Tag())
		}
		return tags
	}
	require.Equal(t, []string{"p2"}, tags(speedTestTargets(lowest, lookup, rests, member)))
	require.Equal(t, []string{"p1", "p2"}, tags(speedTestTargets(selector, lookup, rests, member)))
	require.Equal(t, []string{"p1"}, tags(speedTestTargets(p1, lookup, rests, member)))
	// a group that stands for nothing yet: its proxies in use
	require.Equal(t, []string{"p1", "p2"}, tags(speedTestTargets(lowest, lookup, rests, func(group string) string { return group })))
}
