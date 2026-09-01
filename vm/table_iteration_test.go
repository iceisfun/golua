package vm

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
)

// This file covers what a traversal is allowed to know and allowed to write.
//
// next() resolves the key it is handed to an ordered-keys slot, and on a big
// table it uses the slot index to do it. The index is an accelerator only, so
// every property here has two halves: the answer must be the same with and
// without one, and building or maintaining one must not show up on a table
// that is only ever filled.

// withKeyIndexThreshold forces the slot-lookup path under test: 0 always
// indexes, a huge value always scans.
func withKeyIndexThreshold(t *testing.T, n int) {
	t.Helper()
	old := keyIndexThreshold
	keyIndexThreshold = n
	t.Cleanup(func() { keyIndexThreshold = old })
}

// walkAll returns the keys a full traversal visits, in order, failing rather
// than hanging if the walk revisits a key.
func walkAll(t *testing.T, tbl *Table) []string {
	t.Helper()
	limit := 8*(len(tbl.keys)+len(tbl.array)) + 64
	var out []string
	k, _, err := tbl.Next(Nil)
	for !k.IsNil() {
		if err != nil {
			t.Fatalf("Next: %v", err)
		}
		out = append(out, k.String())
		if len(out) > limit {
			t.Fatalf("traversal did not terminate after %d keys", len(out))
		}
		k, _, err = tbl.Next(k)
	}
	return out
}

// A table that is only ever filled and read must never build a slot index: the
// index is what a traversal pays for, and construction must not pay it.
func TestKeyIndexIsNotBuiltByConstruction(t *testing.T) {
	withKeyIndexThreshold(t, 4)
	tbl := NewEmptyTable()
	for i := 0; i < 200; i++ {
		tbl.SetString(fmt.Sprintf("k%03d", i), NewInt(int64(i)))
	}
	for i := 0; i < 200; i++ {
		tbl.SetInt(1000+i, NewInt(int64(i)))
	}
	for i := 0; i < 100; i++ {
		tbl.SetString(fmt.Sprintf("k%03d", i), Nil)
	}
	for i := 0; i < 100; i++ {
		tbl.SetString(fmt.Sprintf("k%03d", i), NewInt(int64(i)))
	}
	if tbl.GetString("k000").IsNil() {
		t.Fatal("lost a key")
	}
	if x := tbl.keyTracking(); x != nil {
		t.Fatalf("filling a table built a slot index (%d string slots)", len(x.str))
	}
}

// The index appears only when a traversal actually needs it: above the
// threshold, and only once the cursor hint has stopped answering because more
// than one walk is live over the table. A single walk — however long, however
// often repeated — must not build one, because it never has to look a key up.
func TestKeyIndexIsBuiltOnlyWhenTheHintStopsAnswering(t *testing.T) {
	withKeyIndexThreshold(t, 16)

	fill := func(n int) *Table {
		tbl := NewEmptyTable()
		for i := 0; i < n; i++ {
			tbl.SetString(fmt.Sprintf("k%03d", i), NewInt(int64(i)))
		}
		return tbl
	}

	small := fill(8)
	walkAll(t, small)
	if small.keyTracking() != nil {
		t.Fatal("a short keys slice should stay index-free")
	}

	solo := fill(64)
	if solo.keyTracking() != nil {
		t.Fatal("index built before the table was ever traversed")
	}
	for i := 0; i < 3; i++ {
		if got := len(walkAll(t, solo)); got != 64 {
			t.Fatalf("walk %d saw %d keys, want 64", i, got)
		}
		if solo.keyTracking() != nil {
			t.Fatalf("a single sequential walk built an index (walk %d)", i)
		}
	}
	solo.SetString("late", NewInt(1))
	walkAll(t, solo)
	if solo.keyTracking() != nil {
		t.Fatal("walk / insert / walk built an index")
	}

	nested := fill(64)
	outer, _, err := nested.Next(Nil)
	if err != nil {
		t.Fatal(err)
	}
	walkAll(t, nested) // an inner walk displaces the outer walk's hint
	if _, _, err := nested.Next(outer); err != nil {
		t.Fatalf("outer walk could not resume: %v", err)
	}
	if nested.keyTracking() == nil {
		t.Fatal("a displaced traversal should have built the index")
	}
}

// The two lookup paths must agree on everything a traversal can observe.
func TestIndexedAndScannedTraversalsAgree(t *testing.T) {
	build := func() *Table {
		tbl := NewEmptyTable()
		for i := 0; i < 40; i++ {
			tbl.SetString(fmt.Sprintf("s%02d", i), NewInt(int64(i)))
		}
		for i := 40; i > 0; i -= 3 {
			tbl.Set(NewInt(int64(i)), NewInt(int64(i)))
		}
		for i := 0; i < 40; i += 4 {
			tbl.SetString(fmt.Sprintf("s%02d", i), Nil)
		}
		for i := 0; i < 12; i++ {
			tbl.SetString(fmt.Sprintf("late%02d", i), NewInt(int64(i)))
		}
		return tbl
	}

	var orders [2][]string
	for i, thr := range []int{1 << 30, 0} {
		func() {
			withKeyIndexThreshold(t, thr)
			orders[i] = walkAll(t, build())
		}()
	}
	if fmt.Sprint(orders[0]) != fmt.Sprint(orders[1]) {
		t.Fatalf("scanned order %v != indexed order %v", orders[0], orders[1])
	}
}

// Every mutation that can move a key between slots must leave the index
// describing the slice it actually has. checkInvariants audits the index
// against t.keys, so a stale entry fails here rather than silently skipping a
// key in some later traversal.
func TestKeyIndexSurvivesMutation(t *testing.T) {
	enableTableDebugChecks(t)
	withKeyIndexThreshold(t, 4)

	rng := rand.New(rand.NewSource(11))
	tbl := NewEmptyTable()
	live := map[string]bool{}
	for step := 0; step < 4000; step++ {
		switch rng.Intn(6) {
		case 0, 1, 2:
			s := fmt.Sprintf("k%02d", rng.Intn(40))
			tbl.SetString(s, NewInt(int64(step)))
			live[s] = true
		case 3:
			s := fmt.Sprintf("k%02d", rng.Intn(40))
			tbl.SetString(s, Nil)
			delete(live, s)
		case 4:
			// Integer writes, which can promote hash keys into the array.
			tbl.Set(NewInt(int64(1+rng.Intn(20))), NewInt(int64(step)))
		default:
			walkAll(t, tbl)
		}
		checkTable(t, tbl, fmt.Sprintf("step %d", step))
	}
	got := map[string]bool{}
	for _, k := range walkAll(t, tbl) {
		got[k] = true
	}
	for s := range live {
		if !got[s] {
			t.Fatalf("key %q is in the table but a fresh traversal missed it", s)
		}
	}
}

// A key handed back to next() must be accepted whenever the table holds it,
// however the table got that way. An earlier traversal that was abandoned —
// the ordinary "is this table empty?" probe — used to leave a window behind
// that made next() reject a key added afterwards.
func TestNextAcceptsKeyAddedAfterAbandonedProbe(t *testing.T) {
	tbl := NewEmptyTable()
	tbl.SetString("a", NewInt(1))

	if k, _, err := tbl.Next(Nil); err != nil || k.IsNil() {
		t.Fatalf("probe: %v %v", k, err)
	}
	tbl.SetString("b", NewInt(2))

	if _, _, err := tbl.Next(NewString("b")); err != nil {
		t.Fatalf("Next(%q) on a key the table holds: %v", "b", err)
	}
	if _, _, err := tbl.Next(NewString("a")); err != nil {
		t.Fatalf("Next(%q) on a key the table holds: %v", "a", err)
	}
	if _, _, err := tbl.Next(NewString("nope")); err == nil {
		t.Fatal("Next on an absent key should still be rejected")
	}
}

// A fresh traversal must see every live key no matter what earlier traversals
// stopped in the middle of, and no matter what was written between them. The
// keys deliberately mix the array part, the integer hash and the string hash,
// and the probes are abandoned at every depth.
func TestFreshTraversalIsCompleteAfterAbandonedProbes(t *testing.T) {
	enableTableDebugChecks(t)
	rng := rand.New(rand.NewSource(3))
	for seed := 0; seed < 500; seed++ {
		tbl := NewEmptyTable()
		want := map[string]bool{}
		set := func(k Value, v Value) {
			tbl.MustSet(k, v)
			if v.IsNil() {
				delete(want, k.String())
			} else {
				want[k.String()] = true
			}
		}
		for op := 0; op < 30; op++ {
			switch rng.Intn(8) {
			case 0, 1, 2:
				set(NewString(fmt.Sprintf("s%d", rng.Intn(8))), NewInt(int64(op+1)))
			case 3, 4:
				set(NewInt(int64(1+rng.Intn(12))), NewInt(int64(op+1)))
			case 5:
				set(NewString(fmt.Sprintf("s%d", rng.Intn(8))), Nil)
			case 6:
				set(NewInt(int64(1+rng.Intn(12))), Nil)
			default:
				steps := rng.Intn(4)
				k, _, err := tbl.Next(Nil)
				for i := 0; i < steps && err == nil && !k.IsNil(); i++ {
					k, _, err = tbl.Next(k)
				}
			}
			checkTable(t, tbl, "after op")
		}
		got := map[string]int{}
		for _, k := range walkAll(t, tbl) {
			got[k]++
		}
		if len(got) != len(want) {
			t.Fatalf("seed %d: traversal saw %v, want %v", seed, got, want)
		}
		for k := range want {
			if got[k] != 1 {
				t.Fatalf("seed %d: key %s visited %d times (saw %v)", seed, k, got[k], got)
			}
		}
	}
}

// Several traversals of one table are in flight at once, each handing its own
// key back. None of them may lose a key to another's position, and none may
// fail to finish. A single shared cursor on the table makes this quadratic at
// best and wrong at worst; per-walker cursors kept on the VM made it lose keys
// outright.
func TestConcurrentInterleavedTraversalsAreIndependent(t *testing.T) {
	for _, thr := range []int{1 << 30, 0} {
		t.Run(fmt.Sprintf("threshold=%d", thr), func(t *testing.T) {
			withKeyIndexThreshold(t, thr)
			for _, walkers := range []int{2, 3, 5, 8, 12} {
				tbl := NewEmptyTable()
				const n = 40
				for i := 0; i < n; i++ {
					tbl.SetString(fmt.Sprintf("k%02d", i), NewInt(int64(i)))
				}
				seen := make([]map[string]int, walkers)
				cur := make([]Value, walkers)
				for w := range cur {
					seen[w] = map[string]int{}
					k, _, err := tbl.Next(Nil)
					if err != nil {
						t.Fatal(err)
					}
					cur[w] = k
					seen[w][k.String()]++
				}
				for step := 0; step < n*walkers+16; step++ {
					done := true
					for w := range cur {
						if cur[w].IsNil() {
							continue
						}
						done = false
						k, _, err := tbl.Next(cur[w])
						if err != nil {
							t.Fatalf("walker %d: %v", w, err)
						}
						cur[w] = k
						if !k.IsNil() {
							seen[w][k.String()]++
						}
					}
					if done {
						break
					}
				}
				for w := range seen {
					if !cur[w].IsNil() {
						t.Fatalf("%d walkers: walker %d never finished", walkers, w)
					}
					if len(seen[w]) != n {
						t.Fatalf("%d walkers: walker %d saw %d of %d keys", walkers, w, len(seen[w]), n)
					}
					for k, c := range seen[w] {
						if c != 1 {
							t.Fatalf("%d walkers: walker %d saw %s %d times", walkers, w, k, c)
						}
					}
				}
			}
		})
	}
}

// Two walks over one table that both grow it as they go must both stop. The
// window that makes a growing traversal finite lives on the table and is
// shared, so the walk that finishes first must not hand the other one an
// unbounded walk over its own insertions.
func TestInterleavedGrowingWalksTerminate(t *testing.T) {
	for _, walkers := range []int{2, 3, 4, 6} {
		tbl := NewEmptyTable()
		tbl.SetString("k1", NewInt(1))
		tbl.SetString("k2", NewInt(2))
		cur := make([]Value, walkers)
		for w := range cur {
			k, _, err := tbl.Next(Nil)
			if err != nil {
				t.Fatal(err)
			}
			cur[w] = k
		}
		const limit = 512
		steps := 0
		for {
			live := false
			for w := range cur {
				if cur[w].IsNil() {
					continue
				}
				k, _, err := tbl.Next(cur[w])
				if err != nil {
					cur[w] = Nil
					continue
				}
				cur[w] = k
				if k.IsNil() {
					continue
				}
				live = true
				tbl.SetString(k.String()+"x", NewInt(1))
			}
			if !live {
				break
			}
			if steps++; steps > limit {
				t.Fatalf("%d interleaved growing walks did not terminate in %d steps", walkers, limit)
			}
		}
	}
}

// A key deleted and created again is appended at the end of the ordered keys,
// so a walk parked on it ends up standing past its own window. That has to
// leave the walk fenced, and fenced by something that does not move: a walk
// refenced against the current length is refenced higher on every step its own
// body appends. Here the mutation is aimed at the key the walk is standing on,
// every step, which is the worst case — the walk must still run out of entries.
func TestWalkOnRecreatedKeyTerminates(t *testing.T) {
	tbl := NewEmptyTable()
	for i := 0; i < 20; i++ {
		tbl.SetString(fmt.Sprintf("g%d", i), NewInt(int64(i)))
	}
	k, _, err := tbl.Next(Nil)
	if err != nil {
		t.Fatal(err)
	}
	const limit = 256
	steps := 0
	for !k.IsNil() {
		if steps++; steps > limit {
			t.Fatalf("walk over re-created keys did not terminate in %d steps", limit)
		}
		// Delete the key the walk is standing on, hand its slot to something
		// else, then bring it back — which puts it at the end of the slice.
		tbl.Set(k, Nil)
		tbl.SetString(fmt.Sprintf("f%d", steps), NewInt(1))
		tbl.Set(k, NewInt(1))
		tbl.SetString(fmt.Sprintf("a%d", steps), NewInt(1))
		tbl.SetString(fmt.Sprintf("b%d", steps), NewInt(1))
		next, _, err := tbl.Next(k)
		if err != nil {
			break
		}
		k = next
	}
}

// Read-only traversal of a shared table must not write to it. Run under -race
// this is the whole point of resolving a key to its slot instead of caching a
// cursor on the table; without the race detector it still checks that four
// goroutines all see every key.
func TestConcurrentReadOnlyTraversalsDoNotRace(t *testing.T) {
	tbl := NewEmptyTable()
	const n = 64
	for i := 0; i < n; i++ {
		tbl.SetString(fmt.Sprintf("k%02d", i), NewInt(int64(i)))
	}
	// Build the slot index before the goroutines start so the steady-state
	// walk is what is under test; the concurrent-build path is exercised by
	// the shared-table test in the root package.
	walkAll(t, tbl)

	const goroutines = 4
	counts := make([]int, goroutines)
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for r := 0; r < 200; r++ {
				k, _, err := tbl.Next(Nil)
				for !k.IsNil() && err == nil {
					counts[g]++
					k, _, err = tbl.Next(k)
				}
			}
		}(g)
	}
	wg.Wait()
	for g, c := range counts {
		if c != n*200 {
			t.Fatalf("goroutine %d visited %d entries, want %d", g, c, n*200)
		}
	}
}

// An integer key the array part covers must never also be live in the hash
// part. next() crosses from the array into the hash part with no memory of
// where it was, and that is only correct because such a key is always a
// tombstone: otherwise the crossing would either revisit the hash from the
// start or skip it entirely.
func TestArrayCoveredIntegerKeysAreNeverLiveInHash(t *testing.T) {
	enableTableDebugChecks(t)

	// Fill backwards so every key lands in the hash part first, then let the
	// final store promote the whole run into the array.
	tbl := NewEmptyTable()
	for i := 12; i >= 1; i-- {
		tbl.MustSet(NewInt(int64(i)), NewInt(int64(i)))
		checkTable(t, tbl, fmt.Sprintf("after t[%d]", i))
	}
	if got := walkAll(t, tbl); len(got) != 12 {
		t.Fatalf("backward fill traversal saw %v", got)
	}

	// A key that is already in the hash part when the array grows up to it
	// keeps its slot instead of being copied into the array.
	tbl2 := NewEmptyTable()
	tbl2.MustSet(NewInt(5), NewInt(50))
	tbl2.MustSet(NewInt(9), NewInt(90))
	for i := 1; i <= 4; i++ {
		tbl2.MustSet(NewInt(int64(i)), NewInt(int64(i)))
		checkTable(t, tbl2, fmt.Sprintf("after t[%d]", i))
	}
	seen := map[string]int{}
	for _, k := range walkAll(t, tbl2) {
		seen[k]++
	}
	for _, k := range []string{"1", "2", "3", "4", "5", "9"} {
		if seen[k] != 1 {
			t.Fatalf("key %s visited %d times (saw %v)", k, seen[k], seen)
		}
	}
}

// Promotion into the array must not renumber the slots behind it. Splicing the
// promoted key out shifted every later slot, which cost a copy per key and
// threw away the whole slot index; the traversal order either side of the
// promotion has to stay exactly what it was.
func TestPromotionKeepsRemainingSlotOrder(t *testing.T) {
	enableTableDebugChecks(t)
	tbl := NewEmptyTable()
	tbl.SetString("a", NewInt(1))
	tbl.MustSet(NewInt(3), NewInt(3))
	tbl.SetString("b", NewInt(2))
	tbl.MustSet(NewInt(2), NewInt(2))
	tbl.SetString("c", NewInt(3))

	before := walkAll(t, tbl)
	tbl.MustSet(NewInt(1), NewInt(1)) // appends 1, drains 2 and 3 into the array
	checkTable(t, tbl, "after promotion")

	after := walkAll(t, tbl)
	if fmt.Sprint(after) != "[1 2 3 a b c]" {
		t.Fatalf("order after promotion = %v (was %v)", after, before)
	}
	// Every string key keeps its relative order.
	var strs []string
	for _, k := range after {
		if k == "a" || k == "b" || k == "c" {
			strs = append(strs, k)
		}
	}
	if fmt.Sprint(strs) != "[a b c]" {
		t.Fatalf("string keys reordered by promotion: %v", strs)
	}
}

// Promotion tombstones a slot instead of splicing it out, so a fill that keeps
// promoting must not leave the ordered-keys slice growing without bound. The
// two shapes below are the ones that promote every key: a descending fill,
// whose promotions all land on the tail, and an interleaved one, whose
// promotions land in the middle and are cleared once nothing live is left.
func TestPromotionDoesNotGrowOrderedKeys(t *testing.T) {
	enableTableDebugChecks(t)
	const n = 400

	descending := NewEmptyTable()
	for i := n; i >= 1; i-- {
		descending.MustSet(NewInt(int64(i)), NewInt(int64(i)))
	}
	checkTable(t, descending, "descending fill")
	if got := len(descending.keys); got != 0 {
		t.Fatalf("descending fill left %d ordered-keys slots, want 0", got)
	}

	interleaved := NewEmptyTable()
	for i := 2; i <= n; i += 2 {
		interleaved.MustSet(NewInt(int64(i)), NewInt(int64(i)))
	}
	for i := 1; i <= n; i += 2 {
		interleaved.MustSet(NewInt(int64(i)), NewInt(int64(i)))
	}
	checkTable(t, interleaved, "interleaved fill")
	if got := len(interleaved.keys); got != 0 {
		t.Fatalf("interleaved fill left %d ordered-keys slots, want 0", got)
	}
	if got := len(walkAll(t, interleaved)); got != n {
		t.Fatalf("traversal saw %d keys, want %d", got, n)
	}

	// A live string key must keep the slice — and its own slot — intact.
	mixed := NewEmptyTable()
	mixed.SetString("keep", NewInt(1))
	for i := 2; i <= 64; i += 2 {
		mixed.MustSet(NewInt(int64(i)), NewInt(int64(i)))
	}
	for i := 1; i <= 64; i += 2 {
		mixed.MustSet(NewInt(int64(i)), NewInt(int64(i)))
	}
	checkTable(t, mixed, "mixed fill")
	if mixed.GetString("keep").IsNil() {
		t.Fatal("promotion dropped a live string key")
	}
	seen := walkAll(t, mixed)
	if len(seen) != 65 {
		t.Fatalf("traversal saw %d keys, want 65", len(seen))
	}
	if seen[len(seen)-1] != "keep" {
		t.Fatalf("string key lost its position: %v", seen[len(seen)-3:])
	}
}

// Deleting every key must NOT clear the ordered-keys slice: next() has to keep
// finding the tombstones so a traversal parked on a deleted key still ends
// cleanly instead of being told the key was never there.
func TestDeletingEveryKeyKeepsTombstones(t *testing.T) {
	enableTableDebugChecks(t)
	tbl := NewEmptyTable()
	for i := 0; i < 8; i++ {
		tbl.SetString(fmt.Sprintf("k%d", i), NewInt(int64(i)))
	}
	k, _, err := tbl.Next(Nil)
	if err != nil || k.IsNil() {
		t.Fatalf("first step: %v %v", k, err)
	}
	for i := 0; i < 8; i++ {
		tbl.SetString(fmt.Sprintf("k%d", i), Nil)
	}
	checkTable(t, tbl, "after deleting everything")
	nk, _, err := tbl.Next(k)
	if err != nil {
		t.Fatalf("Next after deleting every key: %v", err)
	}
	if !nk.IsNil() {
		t.Fatalf("Next returned %v after every key was deleted", nk)
	}
}
