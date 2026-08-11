package vm

import (
	"fmt"
	"math/rand"
	"testing"
)

// enableTableDebugChecks turns the ordered-keys audit on for the duration of a
// test. The audit is quadratic, so it is off in normal builds.
func enableTableDebugChecks(t *testing.T) {
	t.Helper()
	tableDebugChecks = true
	t.Cleanup(func() { tableDebugChecks = false })
}

// checkTable fails the test if the table's ordered-keys invariants are broken
// (duplicate key, or a deadKeys count that disagrees with the storage maps).
func checkTable(t *testing.T, tbl *Table, where string) {
	t.Helper()
	if err := tbl.checkInvariants(); err != nil {
		t.Fatalf("%s: table invariant violated: %v", where, err)
	}
	if tbl.deadKeys < 0 {
		t.Fatalf("%s: deadKeys went negative: %d", where, tbl.deadKeys)
	}
}

// collectPairs walks the table with Next, mirroring what pairs() does. It fails
// the test rather than hanging if traversal revisits keys — before the
// removeLiveKey fix, a duplicated key in t.keys made Next return that key as
// its own successor and pairs() looped forever at 100% CPU.
func collectPairs(t *testing.T, tbl *Table, limit int) map[any]Value {
	t.Helper()
	got := make(map[any]Value)
	k, v, err := tbl.Next(Nil)
	for steps := 0; !k.IsNil(); steps++ {
		if err != nil {
			t.Fatalf("Next returned error: %v", err)
		}
		if steps >= limit {
			t.Fatalf("Next did not terminate after %d steps (duplicate key in traversal)", limit)
		}
		hk := hashKey(k)
		if _, dup := got[hk]; dup {
			t.Fatalf("Next visited key %v twice", hk)
		}
		got[hk] = v
		k, v, err = tbl.Next(k)
	}
	return got
}

// TestPromotionThenReinsertTerminates covers the hash->array promotion combined
// with a delete/re-insert of an unrelated hash key. The promotion used to
// decrement deadKeys for a live key, which disabled tombstone revival and made
// the re-inserted key land in t.keys twice.
func TestPromotionThenReinsertTerminates(t *testing.T) {
	enableTableDebugChecks(t)

	cases := []struct {
		name     string
		churnKey Value
	}{
		{"string key", NewString("x")},
		{"integer key", NewInt(9)},
		{"float key", NewFloat(1.5)},
		{"bool key", True},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tbl := NewEmptyTable()
			// Out-of-order fill: key 2 lands in the hash part, then key 1
			// extends the array and promotes 2 into it.
			tbl.MustSet(NewInt(2), NewString("b"))
			tbl.MustSet(NewInt(1), NewString("a"))
			checkTable(t, tbl, "after promotion")

			tbl.MustSet(tc.churnKey, NewInt(1))
			tbl.MustSet(tc.churnKey, Nil)
			tbl.MustSet(tc.churnKey, NewInt(2))
			checkTable(t, tbl, "after re-insert")

			got := collectPairs(t, tbl, 32)
			want := map[any]Value{
				int64(1):             NewString("a"),
				int64(2):             NewString("b"),
				hashKey(tc.churnKey): NewInt(2),
			}
			if len(got) != len(want) {
				t.Fatalf("pairs visited %d keys, want %d: %v", len(got), len(want), got)
			}
			for k, wv := range want {
				gv, ok := got[k]
				if !ok {
					t.Fatalf("key %v missing from traversal", k)
				}
				if !gv.RawEqual(wv) {
					t.Errorf("key %v = %s, want %s", k, gv.String(), wv.String())
				}
			}
		})
	}
}

// TestRepeatedPromotionKeepsDeadKeysExact drives many promotions in a row (each
// one used to leak a deadKeys decrement) and checks the counter never drifts.
func TestRepeatedPromotionKeepsDeadKeysExact(t *testing.T) {
	enableTableDebugChecks(t)

	tbl := NewEmptyTable()
	// Fill 2..64 into the hash part, then insert 1 so all of them promote.
	for i := int64(2); i <= 64; i++ {
		tbl.MustSet(NewInt(i), NewInt(i))
	}
	tbl.MustSet(NewString("s"), NewString("s"))
	tbl.MustSet(NewString("s"), Nil) // one genuine tombstone
	tbl.MustSet(NewInt(1), NewInt(1))
	checkTable(t, tbl, "after bulk promotion")

	if tbl.deadKeys != 1 {
		t.Fatalf("deadKeys = %d after promoting 63 live keys, want 1", tbl.deadKeys)
	}
	// The tombstone must still be revived rather than duplicated.
	tbl.MustSet(NewString("s"), NewString("s2"))
	checkTable(t, tbl, "after revival")
	if tbl.deadKeys != 0 {
		t.Fatalf("deadKeys = %d after reviving the only tombstone, want 0", tbl.deadKeys)
	}

	got := collectPairs(t, tbl, 256)
	if len(got) != 65 {
		t.Fatalf("pairs visited %d keys, want 65", len(got))
	}
}

// TestEnsureArraySizeAbsorbsHashIndices covers the other way a key could be
// reached twice by one traversal: growing the array part over an index still
// held in the hash part left the value both shadowed by the (nil) array slot
// and visible to pairs().
func TestEnsureArraySizeAbsorbsHashIndices(t *testing.T) {
	enableTableDebugChecks(t)

	tbl := NewEmptyTable()
	tbl.MustSet(NewInt(3), NewString("three")) // hash-resident: array is empty
	tbl.MustSet(NewString("s"), NewString("s"))
	tbl.EnsureArraySize(4)
	checkTable(t, tbl, "after EnsureArraySize")

	if v := tbl.GetInt(3); !v.RawEqual(NewString("three")) {
		t.Fatalf("t[3] = %s after growing the array over it, want \"three\"", v.String())
	}
	got := collectPairs(t, tbl, 16)
	if len(got) != 2 {
		t.Fatalf("pairs visited %d keys, want 2: %v", len(got), got)
	}
}

// TestPromotionThroughInterpreter reproduces the original hang through the
// interpreter's own table opcodes rather than the Go API.
func TestPromotionThroughInterpreter(t *testing.T) {
	enableTableDebugChecks(t)

	sources := []string{
		`local t = {} t[2]="b" t[1]="a" t.x=1 t.x=nil t.x=2 return t`,
		`local t = {} t[2]="b" t[1]="a" t[9]=9 t[9]=nil t[9]=9 return t`,
		`local t = {[2]="b"} t[1]="a" t[0.5]=1 t[0.5]=nil t[0.5]=2 return t`,
		`local t = {} for i=8,1,-1 do t[i]=i end t.a=1 t.a=nil t.b=2 t.a=3 return t`,
	}
	for i, src := range sources {
		t.Run(fmt.Sprintf("case%d", i), func(t *testing.T) {
			res, err := runWithVM(t, New(), src)
			if err != nil {
				t.Fatalf("run error: %v", err)
			}
			tbl, ok := res[0].AsTable().(*Table)
			if !ok {
				t.Fatalf("expected a table result, got %s", res[0].String())
			}
			checkTable(t, tbl, "after chunk")
			collectPairs(t, tbl, 64)
		})
	}
}

// TestTableKeyChurnSoak hammers insert/delete/re-insert/promote with a fixed
// seed and verifies the ordered-keys invariants and a reference key set after
// every single operation.
func TestTableKeyChurnSoak(t *testing.T) {
	enableTableDebugChecks(t)

	rng := rand.New(rand.NewSource(20260810))
	tbl := NewEmptyTable()
	model := make(map[any]Value)

	keyFor := func() Value {
		switch rng.Intn(4) {
		case 0:
			return NewInt(int64(1 + rng.Intn(12))) // array range: drives promotion
		case 1:
			return NewInt(int64(100 + rng.Intn(8))) // always hash-resident
		case 2:
			return NewString(fmt.Sprintf("k%d", rng.Intn(10)))
		default:
			return NewFloat(float64(rng.Intn(10)) + 0.5)
		}
	}

	for step := 0; step < 4000; step++ {
		k := keyFor()
		var v Value
		if rng.Intn(3) == 0 {
			v = Nil // delete (possibly of an absent key)
		} else {
			v = NewInt(int64(step))
		}
		tbl.MustSet(k, v)
		if v.IsNil() {
			delete(model, hashKey(k))
		} else {
			model[hashKey(k)] = v
		}
		checkTable(t, tbl, fmt.Sprintf("step %d (key %v)", step, hashKey(k)))

		// A full traversal every so often: cheap enough at this table size and
		// the only way to catch a key that iteration skips or repeats.
		if step%16 == 0 {
			got := collectPairs(t, tbl, 256)
			if len(got) != len(model) {
				t.Fatalf("step %d: pairs visited %d keys, model has %d", step, len(got), len(model))
			}
			for mk, mv := range model {
				gv, ok := got[mk]
				if !ok {
					t.Fatalf("step %d: key %v missing from traversal", step, mk)
				}
				if !gv.RawEqual(mv) {
					t.Fatalf("step %d: key %v = %s, want %s", step, mk, gv.String(), mv.String())
				}
			}
		}
	}
}
