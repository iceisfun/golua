package golua_test

// Soak / endurance tests: each runs a single workload HARD for ~10 minutes with
// heavy internal churn, checking a stability invariant throughout. These target
// bugs that only emerge over a long-lived VM doing huge numbers of operations —
// state accumulation/corruption (register/stack/upvalue leaks), GC interaction,
// and slow memory leaks — which the one-shot finders cannot reach.
//
// Gated: they only run when GOLUA_SOAK is set (each takes minutes). Duration is
// GOLUA_SOAK_DURATION (Go duration string, default 10m). Run one at a time:
//
//	GOLUA_SOAK=1 go test -run TestSoakChurn -timeout 0 .
//	GOLUA_SOAK=1 GOLUA_SOAK_DURATION=2m go test -run TestSoak -timeout 0 .

import (
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/iceisfun/golua/v2/compiler"
	"github.com/iceisfun/golua/v2/parser"
	"github.com/iceisfun/golua/v2/stdlib"
	"github.com/iceisfun/golua/v2/vm"
)

func soakGate(t *testing.T) time.Duration {
	t.Helper()
	if os.Getenv("GOLUA_SOAK") == "" {
		t.Skip("soak test: set GOLUA_SOAK=1 to run (minutes per test)")
	}
	d := 10 * time.Minute
	if s := os.Getenv("GOLUA_SOAK_DURATION"); s != "" {
		if parsed, err := time.ParseDuration(s); err == nil {
			d = parsed
		}
	}
	return d
}

func soakCompile(t *testing.T, name, src string) *compiler.Proto {
	t.Helper()
	block, err := parser.Parse(name, src)
	if err != nil {
		t.Fatalf("parse %s: %v", name, err)
	}
	p, err := compiler.Compile(name, block)
	if err != nil {
		t.Fatalf("compile %s: %v", name, err)
	}
	return p
}

func heapInuseMB() float64 {
	runtime.GC()
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return float64(ms.HeapInuse) / (1024 * 1024)
}

// TestSoakChurn: one long-lived VM runs the deterministic churn workload over
// and over. The checksum is identical every iteration; any drift indicates
// state corruption/leakage accumulating in the VM over millions of operations.
func TestSoakChurn(t *testing.T) {
	dur := soakGate(t)
	proto := soakCompile(t, "=churn", churnSrc)
	v := vm.New()
	stdlib.Open(v)
	v.SetGlobal("__n", vm.NewInt(50))

	res, err := v.Run(proto)
	if err != nil {
		t.Fatalf("initial run: %v", err)
	}
	want := res[0].AsInt()

	start := time.Now()
	var iters int64
	baselineMB, lastReport := 0.0, time.Now()
	for time.Since(start) < dur {
		res, err := v.Run(proto)
		if err != nil {
			t.Fatalf("iter %d: %v", iters, err)
		}
		if res[0].AsInt() != want {
			t.Fatalf("iter %d: checksum drift %d != %d (state corruption)", iters, res[0].AsInt(), want)
		}
		iters++
		if iters%5000 == 0 {
			mb := heapInuseMB()
			if baselineMB == 0 {
				baselineMB = mb
			}
			// Flag only egregious unbounded growth (a real leak), not GC noise.
			if mb > baselineMB*8 && mb > 512 {
				t.Fatalf("iter %d: heap grew %.0fMB -> %.0fMB on a fixed workload (likely leak)", iters, baselineMB, mb)
			}
			if time.Since(lastReport) > 30*time.Second {
				t.Logf("soak churn: %d iters, heap %.0fMB", iters, mb)
				lastReport = time.Now()
			}
		}
	}
	t.Logf("soak churn done: %d iters in %s", iters, dur)
}

// TestSoakAllocGC: heavy allocation + explicit collectgarbage inside Lua, on a
// long-lived VM. Verifies the workload stays correct and the Go heap stays
// bounded (no slow leak) across the run.
func TestSoakAllocGC(t *testing.T) {
	dur := soakGate(t)
	src := `
local function work()
  local keep = {}
  for i = 1, 2000 do
    local s = string.rep("x", (i % 64) + 1)
    keep[i % 128] = { s, i, {i, i*i}, setmetatable({}, {__index=function() return i end}) }
    if i % 256 == 0 then collectgarbage() end
  end
  local sum = 0
  for k, vv in pairs(keep) do sum = sum + vv[2] end
  return sum
end
return work()
`
	proto := soakCompile(t, "=allocgc", src)
	v := vm.New()
	stdlib.Open(v)

	start := time.Now()
	var iters int64
	baselineMB := 0.0
	for time.Since(start) < dur {
		if _, err := v.Run(proto); err != nil {
			t.Fatalf("iter %d: %v", iters, err)
		}
		iters++
		if iters%2000 == 0 {
			mb := heapInuseMB()
			if baselineMB == 0 {
				baselineMB = mb
			}
			if mb > baselineMB*8 && mb > 512 {
				t.Fatalf("iter %d: heap %.0fMB -> %.0fMB (likely leak)", iters, baselineMB, mb)
			}
		}
	}
	t.Logf("soak alloc/gc done: %d iters in %s", iters, dur)
}

// TestSoakCoroutineStorm: continuously create, drive, and discard coroutines on
// a long-lived VM. Goroutine-backed coroutines that aren't fully drained/closed
// could leak goroutines; assert the goroutine count stays bounded.
func TestSoakCoroutineStorm(t *testing.T) {
	dur := soakGate(t)
	// The SUPPORTED lifecycle: every coroutine is either driven to completion
	// or explicitly closed, so its goroutine is reaped. (Abandoned *suspended*
	// coroutines leak a goroutine — a documented limitation, see
	// TestCoroutineGoroutineLifecycle and wontfix/coroutine-goroutine-leak.)
	src := `
local total = 0
for c = 1, 200 do
  local co = coroutine.create(function(x)
    for i = 1, 10 do x = coroutine.yield(x + i) end
    return x
  end)
  local _, v = coroutine.resume(co, c)
  if c % 2 == 0 then
    while coroutine.status(co) ~= "dead" do _, v = coroutine.resume(co, v) end
  else
    coroutine.close(co)
  end
  total = total + (v or 0)
end
return total
`
	proto := soakCompile(t, "=costorm", src)
	v := vm.New()
	stdlib.Open(v)

	base := runtime.NumGoroutine()
	start := time.Now()
	var iters int64
	for time.Since(start) < dur {
		if _, err := v.Run(proto); err != nil {
			t.Fatalf("iter %d: %v", iters, err)
		}
		iters++
		if iters%2000 == 0 {
			runtime.GC()
			time.Sleep(20 * time.Millisecond) // let abandoned coroutine goroutines unwind
			if g := runtime.NumGoroutine(); g > base+500 {
				t.Fatalf("iter %d: goroutines grew %d -> %d (abandoned-coroutine leak?)", iters, base, g)
			}
		}
	}
	t.Logf("soak coroutine storm done: %d iters in %s, goroutines %d->%d", iters, dur, base, runtime.NumGoroutine())
}

// TestSoakConcurrentVMs: many VMs churning concurrently for the whole duration,
// continuously created and discarded — stresses shared state + GC under
// sustained concurrent load (run with -race for maximum signal).
func TestSoakConcurrentVMs(t *testing.T) {
	dur := soakGate(t)
	const G = 8
	done := make(chan int64, G)
	start := time.Now()
	for g := 0; g < G; g++ {
		go func() {
			var local int64
			proto := soakCompile(t, "=churn", churnSrc)
			for time.Since(start) < dur {
				v := vm.New()
				stdlib.Open(v)
				v.SetGlobal("__n", vm.NewInt(30))
				if _, err := v.Run(proto); err != nil {
					t.Errorf("concurrent soak: %v", err)
					break
				}
				local++
			}
			done <- local
		}()
	}
	var total int64
	for g := 0; g < G; g++ {
		total += <-done
	}
	t.Logf("soak concurrent VMs done: %d total iters across %d workers in %s", total, G, dur)
}
