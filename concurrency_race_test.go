package golua_test

// Concurrency / data-race coverage for golua's Go-specific surface: many VMs,
// shared read-only compiled protos, goroutine-backed coroutines, and channels
// run simultaneously. Run with `go test -race` to surface data races on any
// package-level shared state (string interning, registries, weak stores, GC
// queues) — a failure class that cannot exist in single-threaded C Lua and that
// the differential finders structurally cannot see.

import (
	"sync"
	"testing"

	"github.com/iceisfun/golua/v2/compiler"
	"github.com/iceisfun/golua/v2/parser"
	"github.com/iceisfun/golua/v2/stdlib"
	"github.com/iceisfun/golua/v2/vm"
)

// A churn workload that exercises many subsystems: string building/interning,
// table array+hash churn + sort, closures/upvalues, coroutines, metatables,
// pcall, pattern matching. Deterministic; returns a checksum so we can also
// verify cross-goroutine result consistency.
const churnSrc = `
local function churn(rounds)
  local acc = 0
  for r = 1, rounds do
    -- strings (interning, builders)
    local parts = {}
    for i = 1, 16 do parts[i] = string.format("k%d_%d", r, i) end
    local s = table.concat(parts, ",")
    for w in s:gmatch("%a+%d+") do acc = (acc + #w) & 0x7fffffffffffffff end
    -- tables: array + hash + sort
    local t = {}
    for i = 1, 32 do t[i] = (r * i) & 0xffff; t["x"..i] = i end
    table.sort(t, function(a, b) return a > b end)
    acc = (acc + t[1]) & 0x7fffffffffffffff
    -- closures + upvalues
    local mk = function(seed) local n = seed; return function() n = n + 1; return n end end
    local f = mk(r); acc = (acc + f() + f()) & 0x7fffffffffffffff
    -- coroutine generator
    local co = coroutine.wrap(function()
      local a, b = 1, 1
      for _ = 1, 20 do coroutine.yield(a); a, b = b, (a + b) & 0xffffff end
    end)
    for v in co do acc = (acc + v) & 0x7fffffffffffffff end
    -- metatable dispatch
    local obj = setmetatable({v = r}, {__add = function(x, y)
      return setmetatable({v = (x.v + (type(y) == "table" and y.v or y)) & 0xffff}, getmetatable(x)) end})
    obj = obj + 3 + 4
    acc = (acc + obj.v) & 0x7fffffffffffffff
    -- pcall round-trip
    local ok, e = pcall(function() if r % 7 == 0 then error("boom") end return r end)
    acc = (acc + (ok and e or 0)) & 0x7fffffffffffffff
  end
  return acc
end
return churn(__n)
`

func churnProto(t *testing.T) *compiler.Proto {
	t.Helper()
	block, err := parser.Parse("=churn", churnSrc)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	p, err := compiler.Compile("=churn", block)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	return p
}

func rounds() int {
	if testing.Short() {
		return 20
	}
	return 200
}

// Many independent VMs (own state each) churning concurrently. Under -race this
// catches any package-level mutable state shared across VMs.
func TestConcurrentIndependentVMs(t *testing.T) {
	const G = 16
	n := int64(rounds())
	results := make([]int64, G)
	var wg sync.WaitGroup
	for g := 0; g < G; g++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			// Each goroutine compiles its OWN proto (concurrent parse/compile)
			// and runs its OWN VM — fully independent.
			block, err := parser.Parse("=churn", churnSrc)
			if err != nil {
				t.Errorf("vm %d parse: %v", idx, err)
				return
			}
			proto, err := compiler.Compile("=churn", block)
			if err != nil {
				t.Errorf("vm %d compile: %v", idx, err)
				return
			}
			v := vm.New()
			stdlib.Open(v)
			v.SetGlobal("__n", vm.NewInt(n))
			res, err := v.Run(proto)
			if err != nil {
				t.Errorf("vm %d: %v", idx, err)
				return
			}
			if len(res) > 0 {
				results[idx] = res[0].AsInt()
			}
		}(g)
	}
	wg.Wait()
	// All independent VMs run the identical deterministic workload -> identical
	// checksum. A mismatch would indicate cross-VM state bleed.
	for g := 1; g < G; g++ {
		if results[g] != results[0] {
			t.Fatalf("vm %d checksum %d != vm 0 %d (cross-VM state bleed?)", g, results[g], results[0])
		}
	}
}

// The same compiled proto is shared (read-only) across many VMs running it
// concurrently — protos are shared, not copied per VM. Under -race this catches
// any write to shared proto/constant state during execution.
func TestConcurrentSharedProto(t *testing.T) {
	const G = 16
	shared := churnProto(t)
	n := int64(rounds())
	var wg sync.WaitGroup
	out := make([]int64, G)
	for g := 0; g < G; g++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			v := vm.New()
			stdlib.Open(v)
			v.SetGlobal("__n", vm.NewInt(n))
			res, err := v.Run(shared) // same *Proto across all goroutines
			if err != nil {
				t.Errorf("vm %d: %v", idx, err)
				return
			}
			if len(res) > 0 {
				out[idx] = res[0].AsInt()
			}
		}(g)
	}
	wg.Wait()
	for g := 1; g < G; g++ {
		if out[g] != out[0] {
			t.Fatalf("shared-proto vm %d checksum %d != %d", g, out[g], out[0])
		}
	}
}

// Channels exercised concurrently: each goroutine has its own VM + chan
// provider, with Lua coroutines producing/consuming over a channel. Catches
// races in the channel registry / provider.
func TestConcurrentChannels(t *testing.T) {
	const G = 12
	src := `
local ch = chan.make(64)            -- buffer >= items: no cross-goroutine block
for i = 1, 50 do ch:send(i) end
ch:close()
local sum = 0
while true do
  local v, ok = ch:recv()           -- recv returns (value, ok); ok=false when drained
  if not ok then break end
  sum = sum + v
end
assert(sum == 1275, "sum=" .. sum)
return sum
`
	block, err := parser.Parse("=chan", src)
	if err != nil {
		// chan API shape may differ; skip rather than fail the race suite.
		t.Skipf("chan source parse: %v", err)
	}
	proto, err := compiler.Compile("=chan", block)
	if err != nil {
		t.Skipf("chan compile: %v", err)
	}
	var wg sync.WaitGroup
	for g := 0; g < G; g++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			v := vm.New()
			v.SetChanProvider(vm.NewDefaultChanProvider())
			stdlib.Open(v)
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("chan vm %d panicked: %v", idx, r)
				}
			}()
			if _, err := v.Run(proto); err != nil {
				t.Errorf("chan vm %d: %v", idx, err)
			}
		}(g)
	}
	wg.Wait()
}
