package golua

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/iceisfun/golua/v2/compiler"
	"github.com/iceisfun/golua/v2/parser"
	"github.com/iceisfun/golua/v2/stdlib"
	"github.com/iceisfun/golua/v2/vm"
)

// runIterationLua runs a chunk under an instruction budget and returns its
// output. Every case in this file involves a table being walked and written at
// the same time, which is undefined in Lua but must never mean unbounded work
// in the process embedding the VM — so a regression fails the test instead of
// taking the host's memory with it.
func runIterationLua(t *testing.T, source string) string {
	t.Helper()
	block, err := parser.Parse("test", source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto, err := compiler.Compile("test", block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	v := vm.New(vm.WithCaptureOutput(true), vm.WithLimits(vm.Limits{MaxInstructions: 2000000}))
	stdlib.Open(v)
	if _, err := v.Run(proto); err != nil {
		t.Fatalf("runtime error: %v", err)
	}
	return strings.Join(v.OutputLines(), "\n")
}

// next(t, k) must never reject a key the table holds. An ordinary "is this
// table empty?" probe leaves a traversal half-finished; a key added after it
// still has to be a valid place to resume from.
func TestNextAcceptsKeyAddedAfterEmptinessProbe(t *testing.T) {
	got := runIterationLua(t, `
local t = {a = 1}
if next(t) ~= nil then end
t.b = 2
print(pcall(next, t, 'b'))
print(pcall(next, t, 'a'))
`)
	for i, line := range strings.Split(got, "\n") {
		if strings.Contains(line, "invalid key") {
			t.Fatalf("line %d rejected a key the table holds: %q", i+1, line)
		}
		if !strings.HasPrefix(line, "true") {
			t.Fatalf("line %d = %q, want a successful next()", i+1, line)
		}
	}
}

// A traversal abandoned partway must not cost a later one any keys. The writes
// after the probe deliberately mix array appends, a hash-key delete and a
// re-insert, so the second traversal has to cope with a promoted key, a
// tombstone and a revived slot at once.
func TestFullTraversalAfterAbandonedProbeSeesEveryKey(t *testing.T) {
	got := runIterationLua(t, `
local t = {}
t[11] = 1
t['b'] = 3
t[2]  = 5
t['a'] = 9
t[5]  = 11
do local k = next(t); for _ = 1, 3 do if k == nil then break end; k = next(t, k) end end
t[1] = 14
t[3] = 17
t[5] = nil
t[4] = 20
t[5] = 21
local ks = {}
for k in pairs(t) do ks[#ks+1] = tostring(k) end
table.sort(ks)
print(table.concat(ks, ','))
`)
	const want = "1,11,2,3,4,5,a,b"
	if got != want {
		t.Fatalf("pairs() saw %q, want %q", got, want)
	}
}

// Nested pairs() loops, with the loop that writes at every depth from 1 to 12
// and at the outermost, middle and innermost position. Iteration state kept
// per walker rather than resolved from the key ran out at a fixed nesting
// depth and either lost keys or stopped terminating past it; there is no depth
// at which any of these may stop finishing.
func TestNestedPairsLoopsTerminateAtEveryDepth(t *testing.T) {
	for _, depth := range []int{1, 2, 3, 5, 8, 12} {
		for _, where := range []string{"outer", "middle", "inner"} {
			for _, mutation := range []string{"insert", "delete"} {
				name := fmt.Sprintf("depth=%d/%s/%s", depth, where, mutation)
				t.Run(name, func(t *testing.T) {
					pos := 1
					switch where {
					case "middle":
						pos = (depth + 1) / 2
					case "inner":
						pos = depth
					}
					// Only the level that writes gets more than one key, so
					// the nesting stays deep without the iteration count
					// growing exponentially with it.
					var b strings.Builder
					fmt.Fprintf(&b, "local T = {}\nfor i = 1, %d do T[i] = {a=1} end\nT[%d] = {k1=1, k2=2}\nlocal n = 0\n", depth, pos)
					indent := ""
					for lvl := 1; lvl <= depth; lvl++ {
						fmt.Fprintf(&b, "%sfor k%d in pairs(T[%d]) do\n", indent, lvl, lvl)
						indent += " "
						fmt.Fprintf(&b, "%sn = n + 1\n", indent)
						if lvl == pos {
							if mutation == "insert" {
								fmt.Fprintf(&b, "%sT[%d][k%d .. 'x'] = 1\n", indent, lvl, lvl)
							} else {
								fmt.Fprintf(&b, "%sT[%d][k%d] = nil\n", indent, lvl, lvl)
							}
						}
					}
					for lvl := depth; lvl >= 1; lvl-- {
						indent = indent[:len(indent)-1]
						fmt.Fprintf(&b, "%send\n", indent)
					}
					b.WriteString("print('done')\n")
					if got := runIterationLua(t, b.String()); got != "done" {
						t.Fatalf("output = %q", got)
					}
				})
			}
		}
	}
}

// Several next() walks over one table, all of them growing it. The window that
// makes a growing traversal finite lives on the table and is shared between
// walks, so the walk that runs out first must not leave the others walking
// their own insertions forever.
func TestInterleavedGrowingWalksOverOneTableTerminate(t *testing.T) {
	for _, walkers := range []int{2, 3, 4, 8} {
		t.Run(fmt.Sprintf("walkers=%d", walkers), func(t *testing.T) {
			src := fmt.Sprintf(`
local t = {k1 = 1, k2 = 2}
local W = {}
for i = 1, %d do W[i] = (next(t)) end
local live, n = %d, 0
while live > 0 do
  n = n + 1
  if n > 400 then print('RUNAWAY') return end
  live = 0
  for i = 1, %d do
    if W[i] ~= nil then
      t[tostring(W[i]) .. 'x' .. i] = 1
      local ok, nk = pcall(next, t, W[i])
      W[i] = ok and nk or nil
      if W[i] ~= nil then live = live + 1 end
    end
  end
end
print('done')`, walkers, walkers, walkers)
			if got := runIterationLua(t, src); got != "done" {
				t.Fatalf("output = %q", got)
			}
		})
	}
}

// Overwriting a field is explicitly allowed while a table is being traversed,
// and one walk overwriting a field another walk has just deleted must not cost
// termination. The delete leaves a tombstone, the insert after it takes that
// slot over, and the overwrite then brings the deleted key back at the end of
// the ordered keys — past the window every walk here is fenced by. A walk
// standing there has to be given a fence of its own; left to run against the
// whole slice it is refenced one slot higher every time any of the three walks
// appends, and three lines of Lua take the host's memory.
func TestWalkOnRecreatedKeyStaysFenced(t *testing.T) {
	got := runIterationLua(t, `
local t = {}
for i = 1, 40 do t['g' .. i] = i end
for i = 1, 10 do t[i] = i end
local K, done = {}, {false, false, false}
local n, live = 0, 3
while live > 0 do
  live = 0
  for w = 1, 3 do
    if not done[w] then
      n = n + 1
      if n > 2000 then print('RUNAWAY') return end
      local k = K[w]
      if k ~= nil then
        if w == 1 then t[k] = nil; t['r' .. n] = 1
        elseif w == 2 then t[k] = 99
        else t['a' .. n] = 1 end
      end
      local ok, nk = pcall(next, t, K[w])
      if not ok or nk == nil then done[w] = true else K[w] = nk; live = live + 1 end
    end
  end
end
print('done')`)
	if got != "done" {
		t.Fatalf("output = %q, want \"done\"", got)
	}
}

// Nested read-only traversals of the same table, and of different tables, must
// see every key at every level. A walker-indexed cursor set silently dropped
// half the keys once the nesting outgrew it.
func TestNestedReadOnlyTraversalsSeeEveryKey(t *testing.T) {
	got := runIterationLua(t, `
local T = {}
for i = 1, 5 do T[i] = {} for j = 1, 4 do T[i]['k' .. j] = j end end
local n = 0
for a in pairs(T[1]) do for b in pairs(T[2]) do for c in pairs(T[3]) do
  for d in pairs(T[4]) do for e in pairs(T[5]) do n = n + 1 end end end end end
print(n)
local S = {}
for j = 1, 4 do S['k' .. j] = j end
local m = 0
for a in pairs(S) do for b in pairs(S) do for c in pairs(S) do
  for d in pairs(S) do for e in pairs(S) do m = m + 1 end end end end end
print(m)
`)
	if got != "1024\n1024" {
		t.Fatalf("nested traversals visited %q, want \"1024\\n1024\"", got)
	}
}

// A table shared between independent VMs and only read must be safe to walk
// concurrently. Sharing one read-only configuration table across VMs is an
// ordinary embedding pattern; run under -race this is the test that a
// traversal does not scribble unsynchronised state on the table it walks.
func TestSharedTableTraversedByConcurrentVMs(t *testing.T) {
	shared := vm.NewEmptyTable()
	const keys = 40
	for i := 0; i < keys; i++ {
		shared.SetString(fmt.Sprintf("k%02d", i), vm.NewInt(int64(i)))
	}
	sharedValue := vm.NewTable(shared)

	block, err := parser.Parse("shared", `
local n = 0
for i = 1, 200 do
  for k, v in pairs(shared) do n = n + 1 end
end
return n`)
	if err != nil {
		t.Fatal(err)
	}
	proto, err := compiler.Compile("shared", block)
	if err != nil {
		t.Fatal(err)
	}

	const goroutines = 4
	results := make([]int64, goroutines)
	errs := make([]error, goroutines)
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			m := vm.New()
			stdlib.Open(m)
			m.SetGlobal("shared", sharedValue)
			r, err := m.Run(proto)
			if err != nil {
				errs[g] = err
				return
			}
			if len(r) > 0 {
				results[g], _ = r[0].ToInt()
			}
		}(g)
	}
	wg.Wait()
	for g := range results {
		if errs[g] != nil {
			t.Fatalf("VM %d: %v", g, errs[g])
		}
		if want := int64(keys * 200); results[g] != want {
			t.Fatalf("VM %d visited %d entries, want %d", g, results[g], want)
		}
	}
}
