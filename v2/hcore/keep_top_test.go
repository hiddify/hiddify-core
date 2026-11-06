package hcore

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/hiddify/hiddify-core/v2/config"
	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/hiddify/ipinfo"
	"github.com/stretchr/testify/require"
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
		// keep-top answering: count is every proxy, so the pick drops only the ones that did not answer
		{"answering keeps every one that answered", map[string]int32{"a": 300, "b": timeout, "c": 100}, 3, time.Second, []string{"c", "a"}},
		{"answering waits for every first ping", map[string]int32{"a": 300, "b": 0, "c": 100}, 3, keepTopSettle, nil},
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

func TestPickTopEarly(t *testing.T) {
	timeout := int32(maxPing)
	tests := []struct {
		name          string
		delays, hints map[string]int32
		count         int
		want          []string
	}{
		{"the rest were slower before",
			map[string]int32{"a": 80, "b": 60, "c": 0, "d": 0},
			map[string]int32{"a": 50, "b": 70, "c": 200, "d": 300}, 2, []string{"b", "a"}},
		{"one still to test was faster before",
			map[string]int32{"a": 80, "b": 60, "c": 0},
			map[string]int32{"a": 50, "b": 70, "c": 70}, 2, nil},
		{"as fast as the best still to test",
			map[string]int32{"a": 80, "b": 60, "c": 0},
			map[string]int32{"a": 50, "b": 70, "c": 80}, 2, []string{"b", "a"}},
		{"a never tested one is waited for",
			map[string]int32{"a": 80, "b": 60, "c": 0},
			map[string]int32{"a": 50, "b": 70}, 2, nil},
		{"a never tested one that answered counts",
			map[string]int32{"a": 80, "b": 60, "c": 30, "d": 0},
			map[string]int32{"a": 50, "b": 70, "d": 900}, 2, []string{"c", "b"}},
		{"failed before does not hold it",
			map[string]int32{"a": 80, "b": 60, "c": 0},
			map[string]int32{"a": 50, "b": 70, "c": previousFailed}, 2, []string{"b", "a"}},
		{"fewer than count answered",
			map[string]int32{"a": 80, "b": timeout, "c": 0},
			map[string]int32{"a": 50, "b": 70, "c": 900}, 2, nil},
		{"no count", map[string]int32{"a": 80}, map[string]int32{"a": 50}, 0, nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := pickTopEarly(test.delays, test.hints, test.count); !slices.Equal(got, test.want) {
				t.Fatalf("got %v, want %v", got, test.want)
			}
		})
	}
}

func TestPreviousDelay(t *testing.T) {
	recent, old := time.Now().Add(-time.Hour), time.Now().Add(-48*time.Hour)
	tests := []struct {
		name    string
		history *adapter.URLTestHistory
		want    int32
	}{
		{"none", nil, 0},
		{"cached ping", &adapter.URLTestHistory{Time: recent, Delay: 120, IsFromCache: true}, 120},
		{"cached timeout", &adapter.URLTestHistory{Time: recent, Delay: maxPing, IsFromCache: true}, previousFailed},
		{"never tested", &adapter.URLTestHistory{IsFromCache: true}, 0},
		{"pinged two days ago", &adapter.URLTestHistory{Time: old, Delay: 120, IsFromCache: true}, 0},
		{"timed out two days ago", &adapter.URLTestHistory{Time: old, Delay: maxPing, IsFromCache: true}, 0},
		{"earlier in this run", &adapter.URLTestHistory{Time: recent, Delay: 90}, 90},
		{"failed earlier in this run", &adapter.URLTestHistory{Time: recent, Delay: maxPing}, previousFailed},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := previousDelay(test.history); got != test.want {
				t.Fatalf("got %d, want %d", got, test.want)
			}
		})
	}
}

func TestWithoutTimedOut(t *testing.T) {
	timeout := int32(maxPing)
	tests := []struct {
		name   string
		delays map[string]int32
		want   []string
	}{
		{"one stopped answering", map[string]int32{"a": 100, "b": timeout, "c": 0}, []string{"a", "c"}},
		{"all answer", map[string]int32{"a": 100, "b": 200}, nil},
		{"every one timed out: the whole list comes back instead", map[string]int32{"a": timeout, "b": timeout}, nil},
		{"none tested yet", map[string]int32{"a": 0, "b": timeout}, nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := withoutTimedOut(test.delays); !slices.Equal(got, test.want) {
				t.Fatalf("got %v, want %v", got, test.want)
			}
		})
	}
}

// keep-top answering (-2) is below 0, and the first check took it for "run them all": the core
// never picked, and the proxies that did not answer kept running.
func TestKeepTopAnsweringPicks(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "data"), 0o755))
	require.NoError(t, Setup(&SetupRequest{BasePath: dir, WorkingDir: dir, TempDir: dir, Mode: SetupMode_OLD}, nil))
	settings := strings.TrimSuffix(hotReloadSettings(t, freeTCPPort(t), "warn"), "}") + fmt.Sprintf(`,"keep-top":%d}`, config.KeepTopAnswering)
	_, err := ChangeHiddifySettings(&ChangeHiddifySettingsRequest{HiddifySettingsJson: settings}, false)
	require.NoError(t, err)
	t.Cleanup(func() {
		Stop()
		clearKeepTop() // the next tests start without a pick under way
	})

	_, err = StartService(static.BaseContext, &StartRequest{ConfigContent: profile(socks("proxy-a", 1080), socks("proxy-b", 1081), socks("proxy-c", 1082))})
	require.NoError(t, err)

	keepTop.Lock()
	defer keepTop.Unlock()
	require.True(t, keepTop.picking, "the whole list runs until the ones that answer are known")
	require.True(t, keepTop.answering)
	require.Equal(t, 3, keepTop.count, "every proxy may be kept")
}

// The app sends the settings of another profile to check its config; keep top's own reload then
// took that profile's keep-top, and the proxies it had just stopped all ran again.
func TestKeepTopOwnReloadKeepsItsSettings(t *testing.T) {
	saved := static.HiddifyOptions
	t.Cleanup(func() {
		static.HiddifyOptions = saved
		keepTop.Lock()
		keepTop.ownReload, keepTop.setting, keepTop.settingTags, keepTop.settingCountries = false, 0, nil, nil
		keepTop.Unlock()
	})
	static.HiddifyOptions = config.DefaultHiddifyOptions()
	static.HiddifyOptions.KeepTop = config.KeepTopAnswering
	count, _, _ := keepTopSettings()
	require.Equal(t, config.KeepTopAnswering, count)

	static.HiddifyOptions = config.DefaultHiddifyOptions() // another profile's, checked
	static.HiddifyOptions.KeepTop = -1
	keepTop.Lock()
	keepTop.ownReload = true
	keepTop.Unlock()
	count, _, _ = keepTopSettings()
	require.Equal(t, config.KeepTopAnswering, count, "keep top's own reload keeps the run's keep-top")

	keepTop.Lock()
	keepTop.ownReload = false
	keepTop.Unlock()
	count, _, _ = keepTopSettings()
	require.Equal(t, -1, count, "a start or a reload of the app takes what it sent")
}

func TestKeepTopCountries(t *testing.T) {
	require.Equal(t, []string{"DE", "GB", "US"}, normalizeCountries([]string{"us", " de", "GB", "us", "", "xyz"}))
	require.Nil(t, normalizeCountries(nil))

	require.Equal(t, "FR", proxyCountry("whatever", &adapter.URLTestHistory{IpInfo: &ipinfo.IpInfo{CountryCode: "fr"}}), "where its IP is")
	require.Equal(t, "DE", proxyCountry("🇩🇪 12 - DE - VLESS", nil), "else the flag in its name")
	require.Equal(t, "", proxyCountry("12 - VLESS", &adapter.URLTestHistory{}))

	countries := []string{"DE", "FR"}
	require.False(t, outsideCountries(nil, "US", true), "no countries: every proxy")
	require.False(t, outsideCountries(countries, "DE", true))
	require.True(t, outsideCountries(countries, "US", false), "another country is left out, tested or not")
	require.False(t, outsideCountries(countries, "", false), "not tested yet: it may be of one")
	require.True(t, outsideCountries(countries, "", true), "tested, and still of no known country")
}

// keep-top-countries keeps top on whatever the count: with "all", every proxy of those countries.
func TestKeepTopCountriesPicks(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "data"), 0o755))
	require.NoError(t, Setup(&SetupRequest{BasePath: dir, WorkingDir: dir, TempDir: dir, Mode: SetupMode_OLD}, nil))
	settings := strings.TrimSuffix(hotReloadSettings(t, freeTCPPort(t), "warn"), "}") + `,"keep-top":-1,"keep-top-countries":["de"]}`
	_, err := ChangeHiddifySettings(&ChangeHiddifySettingsRequest{HiddifySettingsJson: settings}, false)
	require.NoError(t, err)
	t.Cleanup(func() {
		Stop()
		clearKeepTop()
	})

	_, err = StartService(static.BaseContext, &StartRequest{ConfigContent: profile(socks("proxy-a", 1080), socks("proxy-b", 1081))})
	require.NoError(t, err)

	keepTop.Lock()
	defer keepTop.Unlock()
	require.True(t, keepTop.picking, "the whole list runs until the ones of those countries are known")
	require.Equal(t, []string{"DE"}, keepTop.countries)
	require.Equal(t, 2, keepTop.count)
}
