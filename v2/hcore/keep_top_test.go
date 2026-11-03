package hcore

import (
	"slices"
	"testing"
	"time"
)

func TestPickTopProxies(t *testing.T) {
	timeout := int32(maxPing)
	tests := []struct {
		name   string
		delays map[string]int32
		count  int
		waited time.Duration
		want   []string
	}{
		{"waits for the first pings", map[string]int32{"a": 100, "b": 0, "c": 0}, 2, time.Second, nil},
		{"all in: the fastest", map[string]int32{"a": 300, "b": 100, "c": timeout, "d": 200}, 2, time.Second, []string{"b", "d"}},
		{"fewer answer than count", map[string]int32{"a": 300, "b": timeout}, 10, time.Second, []string{"a"}},
		{"none answers: none picked", map[string]int32{"a": timeout, "b": timeout}, 1, time.Hour, nil},
		{"enough answered, the rest settle", map[string]int32{"a": 100, "b": 50, "c": 0}, 2, keepTopSettle, []string{"b", "a"}},
		{"not enough answered yet", map[string]int32{"a": 100, "b": 0, "c": 0}, 2, keepTopSettle, nil},
		{"waited long enough for any", map[string]int32{"a": 100, "b": 0, "c": 0}, 2, keepTopMaxWait, []string{"a"}},
		{"ties go by tag", map[string]int32{"b": 100, "a": 100, "c": 100}, 2, time.Second, []string{"a", "b"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := pickTopProxies(test.delays, test.count, test.waited); !slices.Equal(got, test.want) {
				t.Fatalf("got %v, want %v", got, test.want)
			}
		})
	}
}

func TestWithSelectedKeepsTheCount(t *testing.T) {
	tests := []struct {
		name     string
		pick     []string
		count    int
		selected string
		want     []string
	}{
		{"no selection", []string{"a", "b"}, 2, "", []string{"a", "b"}},
		{"selected among the fastest", []string{"a", "b"}, 2, "b", []string{"a", "b"}},
		{"selected takes the slowest's place", []string{"a", "b", "c"}, 3, "x", []string{"a", "b", "x"}},
		{"room left: selected joins", []string{"a"}, 3, "x", []string{"a", "x"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := withSelected(slices.Clone(test.pick), test.count, test.selected); !slices.Equal(got, test.want) {
				t.Fatalf("got %v, want %v", got, test.want)
			}
		})
	}
}
