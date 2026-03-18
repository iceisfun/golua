package vm

import (
	"runtime"
	"testing"
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
	wt.weak.sweep()

	keys := collectNext(wt)
	if len(keys) != 1 {
		t.Errorf("expected 1 alive entry after sweep, got %d", len(keys))
	}
	if keys[0].AsString() != "c" {
		t.Errorf("expected surviving key 'c', got %v", keys[0])
	}
}
