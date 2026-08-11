package vm

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

// This file guards the table optimizations that trade a linear scan for an
// O(1) lookup: the next() cursor, the tombstone index, the co-allocated
// small-table block, and hash->array promotion. Each of them is only sound
// while the ordered-keys invariants documented on Table hold, so every test
// here runs with tableDebugChecks on and audits the table after mutations.

// keyOrder walks the table with Next and returns the keys in iteration order.
func keyOrder(t *testing.T, tbl *Table) []string {
	t.Helper()
	var out []string
	k, _, err := tbl.Next(Nil)
	for steps := 0; !k.IsNil(); steps++ {
		if err != nil {
			t.Fatalf("Next: %v", err)
		}
		if steps > 4*len(tbl.keys)+4*len(tbl.array)+16 {
			t.Fatalf("Next did not terminate (duplicate key in t.keys?)")
		}
		out = append(out, k.String())
		k, _, err = tbl.Next(k)
	}
	return out
}

func wantOrder(t *testing.T, tbl *Table, want ...string) {
	t.Helper()
	got := keyOrder(t, tbl)
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("iteration order = %v, want %v", got, want)
	}
}

// withDeadIndexThreshold forces the tombstone path under test: 0 always uses
// the index, a huge value always uses the original linear scans.
func withDeadIndexThreshold(t *testing.T, n int) {
	t.Helper()
	old := deadIndexThreshold
	deadIndexThreshold = n
	t.Cleanup(func() { deadIndexThreshold = old })
}

// --- OPT 1: next() cursor -------------------------------------------------

// TestNextCursorHandlesNonSequentialProbes checks that the cached keys-slot
// hint never changes what next() returns: the successor of a key must be the
// same whether the call follows the cursor (sequential pairs()) or jumps
// around, and the cursor must survive deletions and insertions between calls.
func TestNextCursorHandlesNonSequentialProbes(t *testing.T) {
	enableTableDebugChecks(t)

	tbl := NewEmptyTable()
	const n = 64
	for i := 0; i < n; i++ {
		tbl.MustSet(NewString(fmt.Sprintf("k%02d", i)), NewInt(int64(i)))
	}
	// Successors recorded by a straight sequential walk.
	succ := map[string]string{}
	seq := keyOrder(t, tbl)
	for i := 0; i+1 < len(seq); i++ {
		succ[seq[i]] = seq[i+1]
	}
	if len(seq) != n {
		t.Fatalf("sequential walk visited %d keys, want %d", len(seq), n)
	}

	// Random-access probes must agree with the sequential successors. Reset
	// iteration first (Next(Nil)) so iterBound covers the whole slice.
	rng := rand.New(rand.NewSource(7))
	for i := 0; i < 500; i++ {
		tbl.MustNext(Nil)
		from := seq[rng.Intn(len(seq)-1)]
		got, _, err := tbl.Next(NewString(from))
		if err != nil {
			t.Fatalf("Next(%q): %v", from, err)
		}
		if got.String() != succ[from] {
			t.Fatalf("Next(%q) = %q, want %q", from, got.String(), succ[from])
		}
	}

	// A key that is not in the table must still be rejected, cursor or not.
	tbl.MustNext(Nil)
	if _, _, err := tbl.Next(NewString("nope")); err == nil {
		t.Fatal("Next with an absent key should error")
	}
}

// TestNextCursorSurvivesMutationDuringTraversal covers deleting the current
// key mid-walk (allowed by Lua) with the cursor pointing at it.
func TestNextCursorSurvivesMutationDuringTraversal(t *testing.T) {
	enableTableDebugChecks(t)

	tbl := NewEmptyTable()
	const n = 50
	for i := 0; i < n; i++ {
		tbl.MustSet(NewString(fmt.Sprintf("k%02d", i)), NewInt(int64(i)))
	}
	seen := 0
	k, _, err := tbl.Next(Nil)
	for !k.IsNil() {
		if err != nil {
			t.Fatalf("Next: %v", err)
		}
		seen++
		cur := k
		k, _, err = tbl.Next(k)
		tbl.MustSet(cur, Nil) // delete the key we just left
		checkTable(t, tbl, "during traversal")
		if seen > n {
			t.Fatal("traversal did not terminate")
		}
	}
	if seen != n {
		t.Fatalf("traversal visited %d keys, want %d", seen, n)
	}
}

// TestPairsTraversalScalesLinearly is the regression guard for the O(n^2)
// pairs() walk: a full traversal used to rescan t.keys for every key. The
// thresholds are deliberately loose (quadratic growth is 4x per doubling and
// took tens of seconds at 160k keys, linear is milliseconds).
func TestPairsTraversalScalesLinearly(t *testing.T) {
	if testing.Short() {
		t.Skip("scaling probe")
	}
	walk := func(n int) time.Duration {
		tbl := NewEmptyTable()
		for i := 0; i < n; i++ {
			tbl.MustSet(NewString(fmt.Sprintf("key_%d", i)), NewInt(int64(i)))
		}
		start := time.Now()
		count := 0
		k, _, _ := tbl.Next(Nil)
		for !k.IsNil() {
			count++
			k, _, _ = tbl.Next(k)
		}
		d := time.Since(start)
		if count != n {
			t.Fatalf("traversal visited %d of %d keys", count, n)
		}
		return d
	}
	small := walk(20000)
	large := walk(80000)
	// 4x the keys must not cost anywhere near 16x the time. Allow a very wide
	// margin (8x) so a loaded host or a GC pause cannot make this flaky.
	if large > 8*small+50*time.Millisecond {
		t.Fatalf("pairs() traversal looks superlinear: 20k keys %v, 80k keys %v", small, large)
	}
}

// --- OPT 2: tombstone index -----------------------------------------------

// The three observable behaviours of tombstone reuse, each checked on both the
// index-free path and the indexed path so the optimization cannot silently
// change pairs() order.

func TestTombstoneRevivalKeepsOriginalSlot(t *testing.T) {
	enableTableDebugChecks(t)
	for _, thr := range []int{1 << 30, 0} {
		t.Run(fmt.Sprintf("threshold=%d", thr), func(t *testing.T) {
			withDeadIndexThreshold(t, thr)
			tbl := NewEmptyTable()
			for _, s := range []string{"a", "b", "c", "d"} {
				tbl.MustSet(NewString(s), True)
			}
			tbl.MustSet(NewString("b"), Nil)
			tbl.MustSet(NewString("c"), Nil)
			checkTable(t, tbl, "after deletes")
			// Reviving c must put it back in its own slot, after (dead) b.
			tbl.MustSet(NewString("c"), True)
			checkTable(t, tbl, "after revival")
			wantOrder(t, tbl, "a", "c", "d")
			// A fresh key then takes b's slot, ahead of c.
			tbl.MustSet(NewString("x"), True)
			checkTable(t, tbl, "after fresh key")
			wantOrder(t, tbl, "a", "x", "c", "d")
		})
	}
}

func TestTombstoneReuseTakesFirstDeadSlot(t *testing.T) {
	enableTableDebugChecks(t)
	for _, thr := range []int{1 << 30, 0} {
		t.Run(fmt.Sprintf("threshold=%d", thr), func(t *testing.T) {
			withDeadIndexThreshold(t, thr)
			tbl := NewEmptyTable()
			for _, s := range []string{"a", "b", "c", "d"} {
				tbl.MustSet(NewString(s), True)
			}
			tbl.MustSet(NewString("b"), Nil)
			tbl.MustSet(NewString("c"), Nil)
			tbl.MustSet(NewString("x"), True)
			tbl.MustSet(NewString("y"), True)
			checkTable(t, tbl, "after reuse")
			wantOrder(t, tbl, "a", "x", "y", "d")
			if len(tbl.keys) != 4 {
				t.Fatalf("keys slice grew to %d slots, want 4", len(tbl.keys))
			}
		})
	}
}

func TestTombstoneReuseIsCrossType(t *testing.T) {
	enableTableDebugChecks(t)
	for _, thr := range []int{1 << 30, 0} {
		t.Run(fmt.Sprintf("threshold=%d", thr), func(t *testing.T) {
			withDeadIndexThreshold(t, thr)
			tbl := NewEmptyTable()
			tbl.MustSet(NewString("a"), True)
			tbl.MustSet(NewInt(100), True) // outside array range -> hash part
			tbl.MustSet(NewString("b"), True)
			tbl.MustSet(NewInt(100), Nil)
			tbl.MustSet(NewString("x"), True) // string key takes the int slot
			checkTable(t, tbl, "string over int")
			wantOrder(t, tbl, "a", "x", "b")

			tbl2 := NewEmptyTable()
			tbl2.MustSet(NewString("a"), True)
			tbl2.MustSet(NewString("b"), True)
			tbl2.MustSet(NewString("a"), Nil)
			tbl2.MustSet(NewInt(200), True) // int key takes the string slot
			checkTable(t, tbl2, "int over string")
			wantOrder(t, tbl2, "200", "b")
		})
	}
}

// TestTombstoneFloatIntNormalization pins that a float key with an integral
// value revives the integer key's tombstone: both normalize to the same
// storage key, so they must share one t.keys slot.
func TestTombstoneFloatIntNormalization(t *testing.T) {
	enableTableDebugChecks(t)
	for _, thr := range []int{1 << 30, 0} {
		t.Run(fmt.Sprintf("threshold=%d", thr), func(t *testing.T) {
			withDeadIndexThreshold(t, thr)
			tbl := NewEmptyTable()
			tbl.MustSet(NewInt(100), True)
			tbl.MustSet(NewString("z"), True)
			tbl.MustSet(NewFloat(100.0), Nil) // deletes t[100]
			if !tbl.Get(NewInt(100)).IsNil() {
				t.Fatal("t[100.0] = nil should delete t[100]")
			}
			tbl.MustSet(NewInt(100), True)
			checkTable(t, tbl, "after revival")
			wantOrder(t, tbl, "100", "z")
		})
	}
}

// TestTombstoneIndexMatchesLinearScan is the differential between the two
// tombstone paths: the same operation sequence, replayed with the index forced
// on and forced off, must produce identical iteration order, identical
// contents and an identical keys-slice length. Anything the index gets wrong
// about *which* slot to reuse shows up here.
func TestTombstoneIndexMatchesLinearScan(t *testing.T) {
	enableTableDebugChecks(t)

	type op struct {
		key Value
		del bool
	}
	rng := rand.New(rand.NewSource(20260811))
	// A fixed pool of table keys, so both replays use identical key identities
	// (table keys hash by pointer).
	tableKeys := make([]Value, 12)
	for i := range tableKeys {
		tableKeys[i] = NewTable(NewEmptyTable())
	}
	ops := make([]op, 6000)
	for i := range ops {
		var k Value
		switch rng.Intn(5) {
		case 0:
			k = NewString(fmt.Sprintf("s%d", rng.Intn(120)))
		case 1:
			k = NewInt(int64(1000 + rng.Intn(120))) // always hash-resident
		case 2:
			k = NewFloat(float64(rng.Intn(60)) + 0.25)
		case 3:
			k = tableKeys[rng.Intn(len(tableKeys))]
		default:
			k = NewBool(rng.Intn(2) == 0)
		}
		ops[i] = op{key: k, del: rng.Intn(3) == 0}
	}

	replay := func(thr int) ([]string, int, int) {
		old := deadIndexThreshold
		deadIndexThreshold = thr
		defer func() { deadIndexThreshold = old }()
		tbl := NewEmptyTable()
		for i, o := range ops {
			if o.del {
				tbl.MustSet(o.key, Nil)
			} else {
				tbl.MustSet(o.key, NewInt(int64(i)))
			}
			if err := tbl.checkInvariants(); err != nil {
				t.Fatalf("threshold %d, op %d: %v", thr, i, err)
			}
		}
		return keyOrder(t, tbl), len(tbl.keys), tbl.deadKeys
	}

	wantKeys, wantLen, wantDead := replay(1 << 30)
	gotKeys, gotLen, gotDead := replay(0)
	if fmt.Sprint(gotKeys) != fmt.Sprint(wantKeys) {
		t.Fatalf("indexed path iterates in a different order than the linear scan\n got %v\nwant %v", gotKeys, wantKeys)
	}
	if gotLen != wantLen || gotDead != wantDead {
		t.Fatalf("indexed path: %d slots / %d dead, linear scan: %d slots / %d dead", gotLen, gotDead, wantLen, wantDead)
	}
	if len(wantKeys) == 0 {
		t.Fatal("replay produced an empty table; the differential proves nothing")
	}
}

// TestTombstoneIndexFreedWhenRevived checks the index does not outlive the
// tombstones it tracks (it would otherwise pin dead string keys forever).
func TestTombstoneIndexFreedWhenRevived(t *testing.T) {
	enableTableDebugChecks(t)
	withDeadIndexThreshold(t, 0)

	tbl := NewEmptyTable()
	for i := 0; i < 40; i++ {
		tbl.MustSet(NewString(fmt.Sprintf("k%02d", i)), True)
	}
	tbl.MustSet(NewString("k05"), Nil)
	tbl.MustSet(NewString("k07"), Nil)
	tbl.MustSet(NewString("k05"), True)
	if tbl.deadTracking() == nil {
		t.Fatal("index should exist while a tombstone remains")
	}
	tbl.MustSet(NewString("k07"), True)
	checkTable(t, tbl, "after both revivals")
	if tbl.deadKeys != 0 {
		t.Fatalf("deadKeys = %d, want 0", tbl.deadKeys)
	}
	if tbl.deadTracking() != nil {
		t.Fatal("tombstone index outlived the last tombstone")
	}
}

// TestTombstoneChurnScalesLinearly is the regression guard for the O(n^2)
// delete-then-refill blowup. Inserting n fresh keys into a table holding n
// tombstones used to rescan t.keys per insert.
func TestTombstoneChurnScalesLinearly(t *testing.T) {
	if testing.Short() {
		t.Skip("scaling probe")
	}
	churn := func(n int) time.Duration {
		tbl := NewEmptyTable()
		for i := 0; i < n; i++ {
			tbl.MustSet(NewString(fmt.Sprintf("k_%d", i)), NewInt(int64(i)))
		}
		for i := 0; i < n; i++ {
			tbl.MustSet(NewString(fmt.Sprintf("k_%d", i)), Nil)
		}
		start := time.Now()
		for i := 0; i < n; i++ {
			tbl.MustSet(NewString(fmt.Sprintf("fresh_%d", i)), NewInt(int64(i)))
		}
		d := time.Since(start)
		if tbl.deadKeys != 0 {
			t.Fatalf("n=%d: %d tombstones left after a full refill", n, tbl.deadKeys)
		}
		if len(tbl.keys) != n {
			t.Fatalf("n=%d: keys slice grew to %d slots, want %d", n, len(tbl.keys), n)
		}
		return d
	}
	small := churn(10000)
	large := churn(40000)
	if large > 8*small+50*time.Millisecond {
		t.Fatalf("tombstone refill looks superlinear: 10k keys %v, 40k keys %v", small, large)
	}
}

// --- OPT 3: co-allocated small-table block --------------------------------

// TestCoAllocatedTableIsOneAllocation checks the small-hash-hint constructor
// classes really do fold Table, keys backing and string store into one object.
func TestCoAllocatedTableIsOneAllocation(t *testing.T) {
	for _, nhash := range []int{1, 2} {
		got := testing.AllocsPerRun(200, func() {
			tbl := NewTableWithSize(0, nhash)
			tbl.SetString("value", NewInt(1))
			sink = tbl
		})
		if got > 1 {
			t.Errorf("NewTableWithSize(0, %d) + one string key: %v allocs, want 1", nhash, got)
		}
	}
}

var sink *Table

// TestCoAllocatedTableGrowsOutOfItsBlock exercises the paths that abandon an
// inline backing: more keys than the block holds, and more string keys than
// the inline store holds (which migrates to the map).
func TestCoAllocatedTableGrowsOutOfItsBlock(t *testing.T) {
	enableTableDebugChecks(t)

	for _, nhash := range []int{1, 2} {
		tbl := NewTableWithSize(0, nhash)
		var want []string
		for i := 0; i < skInline+6; i++ {
			s := fmt.Sprintf("k%02d", i)
			tbl.SetString(s, NewInt(int64(i)))
			want = append(want, s)
		}
		checkTable(t, tbl, "after growing out of the block")
		wantOrder(t, tbl, want...)
		for i := 0; i < skInline+6; i++ {
			if v := tbl.GetString(fmt.Sprintf("k%02d", i)); v.AsInt() != int64(i) {
				t.Fatalf("nhash=%d: k%02d = %s", nhash, i, v.String())
			}
		}
	}
}

// TestCoAllocatedBackingIsZeroedOnAbandon is the pinning guard: an inline
// backing lives as long as the Table, so anything left behind in it after the
// slice moves elsewhere stays reachable. Checking the arrays directly is the
// only way to see this — a leak is otherwise invisible until a weak table or a
// finalizer fails to fire.
func TestCoAllocatedBackingIsZeroedOnAbandon(t *testing.T) {
	// keys backing: fill both slots, then force the append realloc.
	b := &tableBlock2{}
	b.t.keys = b.keys[:0:tableBlockSlots]
	b.t.sstr = b.sstr[:0:tableBlockSlots]
	tbl := &b.t
	for i := 0; i < 3; i++ {
		tbl.SetString(fmt.Sprintf("k%d", i), NewInt(int64(i)))
	}
	for i := range b.keys {
		if !b.keys[i].IsNil() {
			t.Fatalf("abandoned inline keys slot %d still holds %s", i, b.keys[i].String())
		}
	}
	// string store: overflow the inline capacity, which grows it to skInline
	// and later migrates the whole set to the map.
	for i := 0; i < skInline+2; i++ {
		tbl.SetString(fmt.Sprintf("s%d", i), NewInt(int64(i)))
	}
	if tbl.sstr != nil {
		t.Fatal("expected the inline store to have migrated to the map")
	}
	for i := range b.sstr {
		if b.sstr[i].name != "" || !b.sstr[i].val.IsNil() {
			t.Fatalf("abandoned inline sstr slot %d still holds %q", i, b.sstr[i].name)
		}
	}
}

// TestWeakModeClearsCoAllocatedBacking pins the most damaging version of the
// same bug: a weak table must hold no strong reference to its former keys, and
// an inline backing that survives the switch would hold every one of them.
func TestWeakModeClearsCoAllocatedBacking(t *testing.T) {
	b := &tableBlock2{}
	b.t.keys = b.keys[:0:tableBlockSlots]
	b.t.sstr = b.sstr[:0:tableBlockSlots]
	tbl := &b.t
	inner := NewEmptyTable()
	tbl.MustSet(NewTable(inner), NewInt(1))
	tbl.MustSet(NewString("s"), NewInt(2))

	mt := NewEmptyTable()
	mt.SetString("__mode", NewString("k"))
	tbl.SetMetatable(mt)

	if tbl.weakBackend() == nil {
		t.Fatal("table did not enter weak mode")
	}
	for i := range b.keys {
		if !b.keys[i].IsNil() {
			t.Fatalf("weak table still references %s from inline keys slot %d", b.keys[i].String(), i)
		}
	}
	for i := range b.sstr {
		if b.sstr[i].name != "" || !b.sstr[i].val.IsNil() {
			t.Fatalf("weak table still references %q from inline sstr slot %d", b.sstr[i].name, i)
		}
	}
	// Contents must have survived the migration into weak storage.
	if v := tbl.Get(NewTable(inner)); v.AsInt() != 1 {
		t.Fatalf("weak table lost its table key: %s", v.String())
	}
	if v := tbl.GetString("s"); v.AsInt() != 2 {
		t.Fatalf("weak table lost its string key: %s", v.String())
	}
}

// --- OPT 4: hash -> array promotion ---------------------------------------

// TestPromotionMovesKeysIntoArray checks the occupancy rule actually fires for
// dense non-sequential fills and leaves the table's contents untouched.
func TestPromotionMovesKeysIntoArray(t *testing.T) {
	enableTableDebugChecks(t)

	fills := map[string]func(tbl *Table){
		"backward": func(tbl *Table) {
			for i := 40; i >= 1; i-- {
				tbl.MustSet(NewInt(int64(i)), NewInt(int64(i*10)))
			}
		},
		"from2": func(tbl *Table) {
			for i := 2; i <= 40; i++ {
				tbl.MustSet(NewInt(int64(i)), NewInt(int64(i*10)))
			}
		},
		"shuffled": func(tbl *Table) {
			rng := rand.New(rand.NewSource(3))
			order := rng.Perm(40)
			for _, i := range order {
				tbl.MustSet(NewInt(int64(i+1)), NewInt(int64((i+1)*10)))
			}
		},
	}
	for name, fill := range fills {
		t.Run(name, func(t *testing.T) {
			tbl := NewEmptyTable()
			fill(tbl)
			checkTable(t, tbl, "after fill")
			if len(tbl.array) < 32 {
				t.Fatalf("array part is %d slots; promotion did not fire", len(tbl.array))
			}
			if len(tbl.intHash) != 0 {
				t.Fatalf("%d integer keys left in the hash part", len(tbl.intHash))
			}
			lo := 1
			if name == "from2" {
				lo = 2
			}
			for i := lo; i <= 40; i++ {
				if v := tbl.GetInt(i); v.AsInt() != int64(i*10) {
					t.Fatalf("t[%d] = %s after promotion", i, v.String())
				}
			}
			seen := keyOrder(t, tbl)
			if len(seen) != 41-lo {
				t.Fatalf("pairs visited %d keys, want %d", len(seen), 41-lo)
			}
		})
	}
}

// TestPromotionDeclinedWhenSparse checks the occupancy rule does not promote a
// sparse table into a huge array part.
func TestPromotionDeclinedWhenSparse(t *testing.T) {
	enableTableDebugChecks(t)

	tbl := NewEmptyTable()
	for i := 1; i <= 64; i++ {
		tbl.MustSet(NewInt(int64(i*1000)), NewInt(int64(i)))
	}
	checkTable(t, tbl, "sparse fill")
	if len(tbl.array) != 0 {
		t.Fatalf("sparse fill grew the array part to %d slots", len(tbl.array))
	}
	if len(tbl.intHash) != 64 {
		t.Fatalf("intHash holds %d keys, want 64", len(tbl.intHash))
	}
}

// TestPromotionKeepsDeadKeysExact covers the interaction with the tombstone
// bookkeeping: promotion drops the keys-slice slots of integer keys in the
// promoted range, including tombstones, and deadKeys must follow exactly.
func TestPromotionKeepsDeadKeysExact(t *testing.T) {
	enableTableDebugChecks(t)
	for _, thr := range []int{1 << 30, 0} {
		t.Run(fmt.Sprintf("threshold=%d", thr), func(t *testing.T) {
			withDeadIndexThreshold(t, thr)
			tbl := NewEmptyTable()
			for i := 40; i >= 2; i-- {
				tbl.MustSet(NewInt(int64(i)), NewInt(int64(i)))
				if i%7 == 0 { // leave tombstones scattered through t.keys
					tbl.MustSet(NewInt(int64(i)), Nil)
				}
				tbl.MustSet(NewString(fmt.Sprintf("s%d", i)), NewInt(int64(i)))
				checkTable(t, tbl, fmt.Sprintf("after i=%d", i))
			}
			tbl.MustSet(NewString("s10"), Nil)
			tbl.MustSet(NewString("s10"), NewInt(-1)) // must revive, not duplicate
			checkTable(t, tbl, "after revival")
			for i := 2; i <= 40; i++ {
				want := int64(i)
				if i%7 == 0 {
					want = 0
				}
				v := tbl.GetInt(i)
				if want == 0 {
					if !v.IsNil() {
						t.Fatalf("t[%d] = %s, want nil", i, v.String())
					}
				} else if v.AsInt() != want {
					t.Fatalf("t[%d] = %s, want %d", i, v.String(), want)
				}
			}
			keys := keyOrder(t, tbl)
			seen := map[string]bool{}
			for _, k := range keys {
				if seen[k] {
					t.Fatalf("key %q visited twice", k)
				}
				seen[k] = true
			}
		})
	}
}

// TestPromotionSoak hammers dense integer fills mixed with deletes,
// re-insertions and other key types, auditing the invariants after every
// operation and the full contents periodically.
func TestPromotionSoak(t *testing.T) {
	enableTableDebugChecks(t)

	rng := rand.New(rand.NewSource(20260811))
	tbl := NewEmptyTable()
	model := map[any]Value{}

	for step := 0; step < 6000; step++ {
		var k Value
		switch rng.Intn(5) {
		case 0, 1, 2:
			k = NewInt(int64(1 + rng.Intn(200))) // dense: drives promotion
		case 3:
			k = NewString(fmt.Sprintf("s%d", rng.Intn(30)))
		default:
			k = NewFloat(float64(rng.Intn(20)) + 0.5)
		}
		v := NewInt(int64(step))
		if rng.Intn(4) == 0 {
			v = Nil
		}
		tbl.MustSet(k, v)
		if v.IsNil() {
			delete(model, hashKey(k))
		} else {
			model[hashKey(k)] = v
		}
		checkTable(t, tbl, fmt.Sprintf("step %d", step))

		if step%97 == 0 {
			got := map[any]Value{}
			kk, vv, err := tbl.Next(Nil)
			for n := 0; !kk.IsNil(); n++ {
				if err != nil {
					t.Fatalf("step %d: Next: %v", step, err)
				}
				if n > 2*len(model)+512 {
					t.Fatalf("step %d: traversal did not terminate", step)
				}
				if _, dup := got[hashKey(kk)]; dup {
					t.Fatalf("step %d: key %v visited twice", step, hashKey(kk))
				}
				got[hashKey(kk)] = vv
				kk, vv, err = tbl.Next(kk)
			}
			if len(got) != len(model) {
				t.Fatalf("step %d: traversal saw %d keys, model has %d", step, len(got), len(model))
			}
			for mk, mv := range model {
				gv, ok := got[mk]
				if !ok {
					t.Fatalf("step %d: key %v missing", step, mk)
				}
				if !gv.RawEqual(mv) {
					t.Fatalf("step %d: key %v = %s, want %s", step, mk, gv.String(), mv.String())
				}
			}
		}
	}
}

// TestPromotionIterationIsDeterministic checks that two identically-built
// tables iterate identically: promotion reorders iteration, but it must not
// make it unpredictable.
func TestPromotionIterationIsDeterministic(t *testing.T) {
	enableTableDebugChecks(t)

	build := func() *Table {
		tbl := NewEmptyTable()
		for i := 60; i >= 1; i -= 2 {
			tbl.MustSet(NewInt(int64(i)), NewInt(int64(i)))
		}
		for i := 2; i <= 60; i += 2 {
			tbl.MustSet(NewInt(int64(i)), NewInt(int64(i)))
		}
		tbl.MustSet(NewString("tail"), True)
		return tbl
	}
	first := keyOrder(t, build())
	for i := 0; i < 8; i++ {
		if got := keyOrder(t, build()); fmt.Sprint(got) != fmt.Sprint(first) {
			t.Fatalf("iteration order is not reproducible:\n%v\n%v", got, first)
		}
	}
}

// ForEach must re-read t.keys each step: a callback that inserts can
// reallocate the backing array, and the co-allocated inline backing is zeroed
// when abandoned. A ranged loop would read the wiped array and silently skip
// every key after the insert.
func TestForEachSurvivesMutationDuringIteration(t *testing.T) {
	for _, nhash := range []int{0, 1, 2, 3} {
		for _, prefill := range []int{2, 3, 4, 6, 8} {
			tbl := NewTableWithSize(0, nhash)
			want := map[string]bool{}
			for i := 0; i < prefill; i++ {
				k := string(rune('a' + i))
				tbl.MustSet(NewString(k), NewInt(int64(i)))
				want[k] = true
			}
			seen := map[string]bool{}
			first := true
			tbl.ForEach(func(k, v Value) bool {
				if first {
					first = false
					tbl.MustSet(NewString("zzz_new"), NewInt(999))
				}
				seen[k.AsString()] = true
				return true
			})
			for k := range want {
				if !seen[k] {
					t.Errorf("nhash=%d prefill=%d: key %q never visited (saw %d of %d)",
						nhash, prefill, k, len(seen), len(want))
				}
			}
		}
	}
}
