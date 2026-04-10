package config

import (
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// withLiveBackgroundFake swaps the underlying terminal query with a
// counting fake for the duration of the test, and restores it on
// cleanup. Also clears any SIMTOOL_THEME_MODE override so the cache
// path is actually exercised.
func withLiveBackgroundFake(t *testing.T, result string) *int64 {
	t.Helper()
	origOverride, hadOverride := os.LookupEnv("SIMTOOL_THEME_MODE")
	_ = os.Unsetenv("SIMTOOL_THEME_MODE")

	var calls int64
	origQuery := queryLiveBackground
	queryLiveBackground = func() string {
		atomic.AddInt64(&calls, 1)
		return result
	}
	ResetLiveModeCache()

	t.Cleanup(func() {
		queryLiveBackground = origQuery
		ResetLiveModeCache()
		if hadOverride {
			_ = os.Setenv("SIMTOOL_THEME_MODE", origOverride)
		} else {
			_ = os.Unsetenv("SIMTOOL_THEME_MODE")
		}
	})

	return &calls
}

func TestDetectTerminalDarkModeLive_CachesWithinTTL(t *testing.T) {
	calls := withLiveBackgroundFake(t, "dark")

	// First call populates the cache; subsequent calls within TTL
	// should not invoke the underlying query.
	for i := 0; i < 5; i++ {
		if !DetectTerminalDarkModeLive() {
			t.Errorf("call %d returned false, want true (dark)", i)
		}
	}

	if got := atomic.LoadInt64(calls); got != 1 {
		t.Errorf("underlying query called %d times, want 1", got)
	}
}

func TestDetectTerminalDarkModeLive_OverrideBypassesCache(t *testing.T) {
	// Swap the underlying query so a missed cache would be detected.
	calls := withLiveBackgroundFake(t, "dark")
	_ = os.Setenv("SIMTOOL_THEME_MODE", "light")
	t.Cleanup(func() { _ = os.Unsetenv("SIMTOOL_THEME_MODE") })

	// Override should short-circuit before both the cache and query.
	for i := 0; i < 3; i++ {
		if DetectTerminalDarkModeLive() {
			t.Errorf("override 'light' returned true, want false")
		}
	}

	if got := atomic.LoadInt64(calls); got != 0 {
		t.Errorf("underlying query called %d times under override, want 0", got)
	}
}

func TestDetectTerminalDarkModeLive_CacheExpires(t *testing.T) {
	calls := withLiveBackgroundFake(t, "dark")

	_ = DetectTerminalDarkModeLive()
	if got := atomic.LoadInt64(calls); got != 1 {
		t.Errorf("after 1st call: query count = %d, want 1", got)
	}

	// Simulate TTL expiry by reaching into the mutex-protected state.
	liveModeMu.Lock()
	liveModeAt = time.Now().Add(-2 * liveModeTTL)
	liveModeMu.Unlock()

	_ = DetectTerminalDarkModeLive()
	if got := atomic.LoadInt64(calls); got != 2 {
		t.Errorf("after expiry: query count = %d, want 2", got)
	}
}

func TestDetectTerminalDarkModeLive_ConcurrentSafe(t *testing.T) {
	// This test is the actual race-detector target. Under `go test
	// -race` (which CI runs), concurrent unsynchronized access to
	// liveModeValue / liveModeSet / liveModeAt would fail the test.
	calls := withLiveBackgroundFake(t, "dark")

	var wg sync.WaitGroup
	const goroutines = 50
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if !DetectTerminalDarkModeLive() {
				t.Error("concurrent call returned false")
			}
		}()
	}
	wg.Wait()

	// Even under 50 concurrent callers, the cache guarantees the
	// underlying query runs at most once (within the TTL window).
	if got := atomic.LoadInt64(calls); got != 1 {
		t.Errorf("concurrent: query count = %d, want 1", got)
	}
}
