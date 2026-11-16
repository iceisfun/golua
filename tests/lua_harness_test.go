package tests

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// TestRunLuaTestReapsSuspendedCoroutines pins that the file runner closes its
// VM, so coroutines a script abandons while suspended do not keep their
// goroutines (and the VM they reference) alive for the rest of the test run.
// It is not parallel: the parallel tests are paused while it runs, so the
// goroutine count only moves with what runLuaTest leaves behind.
func TestRunLuaTestReapsSuspendedCoroutines(t *testing.T) {
	const n = 1000
	file := filepath.Join(t.TempDir(), "abandon.lua")
	src := `
for i = 1, 1000 do
  local co = coroutine.create(function() coroutine.yield() end)
  assert(coroutine.resume(co))
end
`
	if err := os.WriteFile(file, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	before := runtime.NumGoroutine()
	runLuaTest(t, file)

	// Close waits for each reaped coroutine to finish, but its goroutine can
	// still be on its way out when Close returns, so allow it a moment.
	deadline := time.Now().Add(5 * time.Second)
	for {
		after := runtime.NumGoroutine()
		if after-before < n/10 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("goroutines %d -> %d after a file abandoned %d suspended coroutines",
				before, after, n)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
