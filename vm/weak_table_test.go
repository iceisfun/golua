package vm

import (
	"math"
	"runtime"
	"testing"
	"time"
)

// makeWeakTable creates a table with the given __mode and returns it.
func makeWeakTable(mode string) *Table {
	t := NewEmptyTable()
	mt := NewEmptyTable()
	mt.SetString("__mode", NewString(mode))
	t.SetMetatable(mt)
	return t
}

func TestWeakValueTable_CollectedEntry(t *testing.T) {
	wt := makeWeakTable("v")

	// Create a collectable value and store it.
	val := NewEmptyTable()
	wt.SetString("key", NewTable(val))

	// Value is accessible while alive.
	got := wt.GetString("key")
	if got.IsNil() {
		t.Fatal("expected non-nil value while alive")
	}

	// Drop the strong reference and trigger GC.
	val = nil
	_ = val
	runtime.GC()
	runtime.GC()

	// Value should now be collected.
	got = wt.GetString("key")
	if !got.IsNil() {
		t.Errorf("expected nil after collection, got %v", got)
	}
}

func TestWeakKeyTable_CollectedEntry(t *testing.T) {
	wt := makeWeakTable("k")

	// Create a collectable key and store it.
	key := NewEmptyTable()
	wt.MustSet(NewTable(key), NewString("value"))

	// Accessible while key is alive.
	got := wt.Get(NewTable(key))
	if got.AsString() != "value" {
		t.Fatalf("expected 'value', got %v", got)
	}

	// Drop the key and trigger GC.
	key = nil
	_ = key
	runtime.GC()
	runtime.GC()

	// The entry should be gone. Verify via iteration (can't Get with dead key).
	keys := collectNext(wt)
	if len(keys) != 0 {
		t.Errorf("expected 0 keys after collection, got %d", len(keys))
	}
}

func TestWeakValueTable_ValueTypesNotCollected(t *testing.T) {
	wt := makeWeakTable("v")

	// Value types should never be collected.
	wt.SetString("int", NewInt(42))
	wt.SetString("str", NewString("hello"))
	wt.SetString("bool", NewBool(true))
	wt.SetString("float", NewFloat(3.14))

	runtime.GC()
	runtime.GC()

	if wt.GetString("int").AsInt() != 42 {
		t.Error("int value should survive GC")
	}
	if wt.GetString("str").AsString() != "hello" {
		t.Error("string value should survive GC")
	}
	if !wt.GetString("bool").AsBool() {
		t.Error("bool value should survive GC")
	}
	if wt.GetString("float").AsFloat() != 3.14 {
		t.Error("float value should survive GC")
	}
}

func TestWeakKeyTable_ValueTypeKeysNotCollected(t *testing.T) {
	wt := makeWeakTable("k")

	wt.SetString("str", NewString("hello"))
	wt.SetInt(1, NewString("one"))

	runtime.GC()
	runtime.GC()

	if wt.GetString("str").AsString() != "hello" {
		t.Error("string-keyed entry should survive GC")
	}
	if wt.GetInt(1).AsString() != "one" {
		t.Error("int-keyed entry should survive GC")
	}
}

func TestWeakKVTable_BothCollected(t *testing.T) {
	wt := makeWeakTable("kv")

	key := NewEmptyTable()
	val := NewEmptyTable()
	wt.MustSet(NewTable(key), NewTable(val))

	// Alive while both referenced.
	got := wt.Get(NewTable(key))
	if got.IsNil() {
		t.Fatal("expected non-nil value")
	}

	// Drop value only — entry should be removed.
	val = nil
	_ = val
	runtime.GC()
	runtime.GC()

	keys := collectNext(wt)
	if len(keys) != 0 {
		t.Errorf("expected 0 keys after value collection, got %d", len(keys))
	}
}

func TestWeakValueTable_NextSkipsDead(t *testing.T) {
	wt := makeWeakTable("v")

	alive := NewEmptyTable()
	dead := NewEmptyTable()

	wt.SetString("alive", NewTable(alive))
	wt.SetString("dead", NewTable(dead))
	wt.SetString("also_alive", NewInt(123))

	// Drop the dead value.
	dead = nil
	_ = dead
	runtime.GC()
	runtime.GC()

	// Iteration should skip the dead entry.
	keys := collectNext(wt)
	for _, k := range keys {
		if k.AsString() == "dead" {
			t.Error("dead entry should be skipped during iteration")
		}
	}
	if len(keys) != 2 {
		t.Errorf("expected 2 alive entries, got %d", len(keys))
	}
	runtime.KeepAlive(alive)
}

func TestWeakValueTable_Len(t *testing.T) {
	wt := makeWeakTable("v")

	wt.SetInt(1, NewString("one"))
	wt.SetInt(2, NewString("two"))

	if wt.Len() != 2 {
		t.Fatalf("expected len 2, got %d", wt.Len())
	}
}

func TestWeakTable_SetMetatableTransition(t *testing.T) {
	tbl := NewEmptyTable()
	tbl.SetString("key", NewString("value"))
	tbl.SetInt(1, NewString("one"))

	// Transition to weak mode.
	mt := NewEmptyTable()
	mt.SetString("__mode", NewString("v"))
	tbl.SetMetatable(mt)

	// Data should be preserved.
	if tbl.GetString("key").AsString() != "value" {
		t.Error("data lost on transition to weak mode")
	}
	if tbl.GetInt(1).AsString() != "one" {
		t.Error("array data lost on transition to weak mode")
	}

	// Transition back to strong mode.
	tbl.SetMetatable(nil)

	// Data should still be preserved.
	if tbl.GetString("key").AsString() != "value" {
		t.Error("data lost on transition from weak mode")
	}
	if tbl.GetInt(1).AsString() != "one" {
		t.Error("array data lost on transition from weak mode")
	}
}

func TestWeakValueTable_ForEachSkipsDead(t *testing.T) {
	wt := makeWeakTable("v")

	alive := NewEmptyTable()
	dead := NewEmptyTable()

	wt.SetString("alive", NewTable(alive))
	wt.SetString("dead", NewTable(dead))

	dead = nil
	_ = dead
	runtime.GC()
	runtime.GC()

	var count int
	wt.ForEach(func(key, value Value) bool {
		if key.AsString() == "dead" {
			t.Error("ForEach should skip dead entries")
		}
		count++
		return true
	})

	if count != 1 {
		t.Errorf("expected 1 alive entry in ForEach, got %d", count)
	}
	runtime.KeepAlive(alive)
}

func TestWeakTable_DeleteEntry(t *testing.T) {
	wt := makeWeakTable("v")

	val := NewEmptyTable()
	wt.SetString("key", NewTable(val))

	// Delete explicitly.
	wt.Delete(NewString("key"))

	got := wt.GetString("key")
	if !got.IsNil() {
		t.Errorf("expected nil after delete, got %v", got)
	}
}

func TestWeakStore_SweepCleansDeadEntries(t *testing.T) {
	wt := makeWeakTable("v")

	dead1 := NewEmptyTable()
	dead2 := NewEmptyTable()

	wt.SetString("a", NewTable(dead1))
	wt.SetString("b", NewTable(dead2))
	wt.SetString("c", NewInt(42))

	dead1 = nil
	dead2 = nil
	_ = dead1
	_ = dead2
	runtime.GC()
	runtime.GC()

	// Sweep should clean up dead entries.
	wt.weakBackend().sweep()

	keys := collectNext(wt)
	if len(keys) != 1 {
		t.Errorf("expected 1 alive entry after sweep, got %d", len(keys))
	}
	if keys[0].AsString() != "c" {
		t.Errorf("expected surviving key 'c', got %v", keys[0])
	}
}

// TestWeakTable_FloatKeysNormalized checks that a weak table stores a float key
// with an exact integer value as that integer, as a normal table does, so
// next() hands back 2 and 0 rather than 2.0 and -0.0.
func TestWeakTable_FloatKeysNormalized(t *testing.T) {
	for _, mode := range []string{"k", "v", "kv"} {
		wt := makeWeakTable(mode)
		wt.MustSet(NewFloat(2.0), NewString("b"))
		wt.MustSet(NewFloat(math.Copysign(0, -1)), NewString("z"))
		wt.MustSet(NewFloat(2.5), NewString("f"))

		keys := collectNext(wt)
		if len(keys) != 3 {
			t.Fatalf("mode %q: got %d keys, want 3", mode, len(keys))
		}
		for _, k := range keys {
			switch {
			case k.IsInt():
				if i := k.ival(); i != 2 && i != 0 {
					t.Errorf("mode %q: unexpected integer key %v", mode, k)
				}
			case k.IsFloat() && k.fval() == 2.5:
			default:
				t.Errorf("mode %q: key %v not normalized to an integer", mode, k)
			}
		}
		if got := wt.GetInt(2); got.AsString() != "b" {
			t.Errorf("mode %q: t[2] = %v, want b", mode, got)
		}
	}
}

// TestWeakTable_ClearDuringTraversal checks that a field cleared during a
// traversal stays a valid key to next(), as the manual allows, so the
// `for k in pairs(t) do t[k] = nil end` idiom empties a weak table instead of
// failing with "invalid key to 'next'".
func TestWeakTable_ClearDuringTraversal(t *testing.T) {
	for _, mode := range []string{"k", "v", "kv"} {
		wt := makeWeakTable(mode)
		objs := make([]*Table, 8)
		for i := range objs {
			objs[i] = NewEmptyTable()
			wt.MustSet(NewTable(objs[i]), NewInt(int64(i)))
			wt.MustSet(NewInt(int64(i+1)), NewTable(objs[i]))
			wt.MustSet(NewString(string(rune('a'+i))), NewInt(int64(i)))
		}

		visited := 0
		k, _, err := wt.Next(Nil)
		for !k.IsNil() {
			visited++
			wt.MustSet(k, Nil)
			if k, _, err = wt.Next(k); err != nil {
				t.Fatalf("mode %q: next after clearing a field: %v", mode, err)
			}
		}
		if visited != 3*len(objs) {
			t.Errorf("mode %q: traversal visited %d keys, want %d", mode, visited, 3*len(objs))
		}
		if keys := collectNext(wt); len(keys) != 0 {
			t.Errorf("mode %q: %d keys left after clearing, want 0", mode, len(keys))
		}

		// A cleared key can be stored again and is found and traversed.
		wt.MustSet(NewTable(objs[3]), NewInt(30))
		wt.MustSet(NewString("c"), NewInt(3))
		if got := wt.Get(NewTable(objs[3])); !got.IsInt() || got.ival() != 30 {
			t.Errorf("mode %q: revived key reads %v, want 30", mode, got)
		}
		if keys := collectNext(wt); len(keys) != 2 {
			t.Errorf("mode %q: %d keys after re-storing two, want 2", mode, len(keys))
		}
		runtime.KeepAlive(objs)
	}
}

// weakFill is the linear baseline for the weak-table scaling probes: n fresh
// keys stored into an empty weak table.
func weakFill(keys []Value) time.Duration {
	wt := makeWeakTable("k")
	start := time.Now()
	for _, k := range keys {
		wt.MustSet(k, True)
	}
	return time.Since(start)
}

// TestWeakTableChurnScalesLinearly guards a weak registry or cache under
// churn: each store of a new key after a delete reuses the dead slot. Finding
// that slot used to scan entries from the front, so each store paid O(n).
func TestWeakTableChurnScalesLinearly(t *testing.T) {
	churn := func(n int) time.Duration {
		keys := intHashKeys(2 * n)
		wt := makeWeakTable("k")
		for _, k := range keys[:n] {
			wt.MustSet(k, True)
		}
		start := time.Now()
		for i := 0; i < n; i++ {
			wt.MustSet(keys[i], Nil)
			wt.MustSet(keys[n+i], True)
		}
		d := time.Since(start)
		if !wt.Get(keys[n-1]).IsNil() || wt.Get(keys[2*n-1]).IsNil() {
			t.Fatalf("weak table contents are wrong after %d replacements", n)
		}
		if got := len(wt.weakBackend().entries); got != n {
			t.Fatalf("weak table holds %d slots for %d keys", got, n)
		}
		return d
	}
	checkCostIsLinear(t, "weak table churn", 1<<16, churn,
		func(n int) time.Duration { return weakFill(intHashKeys(n)) })
}

// TestWeakTableFreeSlotsStayBounded checks that clearing and re-storing the
// same key, which revives its slot in place, does not grow the free-slot
// stack, and that a slot revived while on the stack is not handed out again.
func TestWeakTableFreeSlotsStayBounded(t *testing.T) {
	wt := makeWeakTable("k")
	for i := 1; i <= 4; i++ {
		wt.SetInt(i, NewInt(int64(i)))
	}
	for r := 0; r < 1000; r++ {
		wt.SetInt(4, Nil)
		wt.SetInt(4, NewInt(int64(r)))
	}
	ws := wt.weakBackend()
	if len(ws.free) > len(ws.entries) {
		t.Fatalf("free stack holds %d slots for %d entries", len(ws.free), len(ws.entries))
	}
	wt.SetInt(2, Nil)
	wt.SetString("x", True)
	wt.SetString("y", True)
	if got := wt.GetInt(4); !got.IsInt() || got.ival() != 999 {
		t.Errorf("t[4] = %v, want 999", got)
	}
	if n := len(collectNext(wt)); n != 5 {
		t.Errorf("weak table has %d keys, want 5", n)
	}
}
