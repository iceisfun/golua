package vm

import "testing"

// collectNext iterates through all key-value pairs using Next.
func collectNext(t LuaTable) []Value {
	var keys []Value
	k, _, _ := t.Next(Nil)
	for !k.IsNil() {
		keys = append(keys, k)
		k, _, _ = t.Next(k)
	}
	return keys
}

// TestNextInsertionOrder verifies Next yields hash keys in insertion order.
func TestNextInsertionOrder(t *testing.T) {
	tbl := NewEmptyTable()
	tbl.SetString("c", NewInt(3))
	tbl.SetString("a", NewInt(1))
	tbl.SetString("b", NewInt(2))

	keys := collectNext(tbl)
	if len(keys) != 3 {
		t.Fatalf("expected 3 keys, got %d", len(keys))
	}
	expect := []string{"c", "a", "b"}
	for i, k := range keys {
		if k.AsString() != expect[i] {
			t.Errorf("key %d: expected %q, got %q", i, expect[i], k.AsString())
		}
	}
}

// TestNextDeterminism verifies repeated iteration yields identical order.
func TestNextDeterminism(t *testing.T) {
	tbl := NewEmptyTable()
	for i := 0; i < 20; i++ {
		tbl.MustSet(NewString(string(rune('A'+i))), NewInt(int64(i)))
	}

	keys1 := collectNext(tbl)
	keys2 := collectNext(tbl)

	if len(keys1) != len(keys2) {
		t.Fatalf("key counts differ: %d vs %d", len(keys1), len(keys2))
	}
	for i := range keys1 {
		if !keys1[i].Equal(keys2[i]) {
			t.Errorf("key %d differs: %v vs %v", i, keys1[i], keys2[i])
		}
	}
}

// TestNextResetOnNil verifies exhausting Next then restarting with nil yields full traversal.
func TestNextResetOnNil(t *testing.T) {
	tbl := NewEmptyTable()
	tbl.SetString("x", NewInt(1))
	tbl.SetString("y", NewInt(2))

	// Exhaust iteration
	keys1 := collectNext(tbl)

	// Restart
	keys2 := collectNext(tbl)

	if len(keys1) != len(keys2) {
		t.Fatalf("restart produced different count: %d vs %d", len(keys1), len(keys2))
	}
	for i := range keys1 {
		if !keys1[i].Equal(keys2[i]) {
			t.Errorf("key %d differs on restart: %v vs %v", i, keys1[i], keys2[i])
		}
	}
}

// TestNextNoDuplicateFinalKey verifies terminal Next returns (nil, nil) without repeating the last key.
func TestNextNoDuplicateFinalKey(t *testing.T) {
	tbl := NewEmptyTable()
	tbl.SetString("only", NewInt(1))

	k, v, err := tbl.Next(Nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if k.AsString() != "only" || v.AsInt() != 1 {
		t.Fatalf("expected (only, 1), got (%v, %v)", k, v)
	}

	k2, v2, err := tbl.Next(k)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !k2.IsNil() || !v2.IsNil() {
		t.Errorf("expected (nil, nil) after last key, got (%v, %v)", k2, v2)
	}

	// Calling Next with a key that is "not found" must return an error
	_, _, err = tbl.Next(NewString("nonexistent"))
	if err == nil {
		t.Errorf("expected error for nonexistent key, but got none")
	}
}

// TestDeleteAndIteration verifies iteration correctness after deleting a key.
func TestDeleteAndIteration(t *testing.T) {
	tbl := NewEmptyTable()
	tbl.SetString("a", NewInt(1))
	tbl.SetString("b", NewInt(2))
	tbl.SetString("c", NewInt(3))

	_ = tbl.Delete(NewString("b"))

	keys := collectNext(tbl)
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys after delete, got %d", len(keys))
	}
	expect := []string{"a", "c"}
	for i, k := range keys {
		if k.AsString() != expect[i] {
			t.Errorf("key %d: expected %q, got %q", i, expect[i], k.AsString())
		}
	}
}

// TestMixedInsertDeleteLifecycle interleaves Set/Delete and verifies Next consistency.
func TestMixedInsertDeleteLifecycle(t *testing.T) {
	tbl := NewEmptyTable()
	tbl.SetString("a", NewInt(1))
	tbl.SetString("b", NewInt(2))
	_ = tbl.Delete(NewString("a"))
	tbl.SetString("c", NewInt(3))
	tbl.SetString("d", NewInt(4))
	_ = tbl.Delete(NewString("c"))

	keys := collectNext(tbl)
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
	expect := []string{"b", "d"}
	for i, k := range keys {
		if k.AsString() != expect[i] {
			t.Errorf("key %d: expected %q, got %q", i, expect[i], k.AsString())
		}
	}
}

// TestArrayAndHashMixedTraversal verifies Next traverses both array and hash parts.
func TestArrayAndHashMixedTraversal(t *testing.T) {
	tbl := NewEmptyTable()
	tbl.SetInt(1, NewString("one"))
	tbl.SetInt(2, NewString("two"))
	tbl.SetString("name", NewString("test"))

	keys := collectNext(tbl)
	if len(keys) != 3 {
		t.Fatalf("expected 3 keys, got %d", len(keys))
	}
	// Array part comes first (1, 2), then hash part ("name")
	if keys[0].AsInt() != 1 {
		t.Errorf("key 0: expected 1, got %v", keys[0])
	}
	if keys[1].AsInt() != 2 {
		t.Errorf("key 1: expected 2, got %v", keys[1])
	}
	if keys[2].AsString() != "name" {
		t.Errorf("key 2: expected 'name', got %v", keys[2])
	}
}

// TestMetatableViaInterface verifies SetMetatable/Metatable round-trip through LuaTable.
func TestMetatableViaInterface(t *testing.T) {
	var tbl LuaTable = NewEmptyTable()
	var mt LuaTable = NewEmptyTable()
	_ = mt.Set(NewString("__index"), NewString("test"))

	tbl.SetMetatable(mt)
	got := tbl.Metatable()
	if got == nil {
		t.Fatal("expected non-nil metatable")
	}
	if v := got.Get(NewString("__index")); v.AsString() != "test" {
		t.Errorf("expected 'test', got %v", v)
	}
}

// TestNilMetatable verifies SetMetatable(nil) clears metatable.
func TestNilMetatable(t *testing.T) {
	tbl := NewEmptyTable()
	mt := NewEmptyTable()
	tbl.SetMetatable(mt)
	if tbl.Metatable() == nil {
		t.Fatal("metatable should not be nil after setting")
	}

	tbl.SetMetatable(nil)
	if tbl.Metatable() != nil {
		t.Error("metatable should be nil after clearing")
	}
}

// TestDeleteMethod verifies Delete removes key and Get returns Nil.
func TestDeleteMethod(t *testing.T) {
	tbl := NewEmptyTable()
	tbl.SetString("key", NewInt(42))
	if tbl.Get(NewString("key")).AsInt() != 42 {
		t.Fatal("key should exist")
	}

	_ = tbl.Delete(NewString("key"))
	if !tbl.Get(NewString("key")).IsNil() {
		t.Error("key should be nil after delete")
	}
}

// TestLenViaInterface verifies Len() matches array part count.
func TestLenViaInterface(t *testing.T) {
	var tbl LuaTable = NewEmptyTable()
	_ = tbl.Set(NewInt(1), NewString("a"))
	_ = tbl.Set(NewInt(2), NewString("b"))
	_ = tbl.Set(NewInt(3), NewString("c"))
	_ = tbl.Set(NewString("name"), NewString("test")) // hash, not counted

	if tbl.Len() != 3 {
		t.Errorf("expected Len() == 3, got %d", tbl.Len())
	}
}

// TestLuaTableEquality verifies two Values wrapping the same *Table are Equal.
func TestLuaTableEquality(t *testing.T) {
	tbl := NewEmptyTable()
	v1 := NewTable(tbl)
	v2 := NewTable(tbl)

	if !v1.Equal(v2) {
		t.Error("two Values wrapping the same table should be equal")
	}

	tbl2 := NewEmptyTable()
	v3 := NewTable(tbl2)
	if v1.Equal(v3) {
		t.Error("two Values wrapping different tables should not be equal")
	}
}

// TestIteratorResetRegression is the Go-level version of gopher-lua #514 regression.
// Two consecutive for-in loops should both yield all keys.
func TestIteratorResetRegression(t *testing.T) {
	tbl := NewEmptyTable()
	tbl.SetString("a", NewInt(1))
	tbl.SetString("b", NewInt(2))
	tbl.SetString("c", NewInt(3))

	// First traversal
	seen1 := make(map[string]bool)
	k, _, _ := tbl.Next(Nil)
	for !k.IsNil() {
		seen1[k.AsString()] = true
		k, _, _ = tbl.Next(k)
	}

	// Second traversal
	seen2 := make(map[string]bool)
	k, _, _ = tbl.Next(Nil)
	for !k.IsNil() {
		seen2[k.AsString()] = true
		k, _, _ = tbl.Next(k)
	}

	for _, key := range []string{"a", "b", "c"} {
		if !seen1[key] {
			t.Errorf("first loop missing key %q", key)
		}
		if !seen2[key] {
			t.Errorf("second loop missing key %q", key)
		}
	}
}

// TestTypedNilMetatableGuard verifies that (*Table)(nil) passed as LuaTable is handled correctly.
func TestTypedNilMetatableGuard(t *testing.T) {
	tbl := NewEmptyTable()
	mt := NewEmptyTable()
	tbl.SetMetatable(mt)

	// Pass a typed nil (*Table)(nil) as LuaTable
	var nilTable *Table
	tbl.SetMetatable(nilTable)
	if tbl.Metatable() != nil {
		t.Error("metatable should be nil after setting typed-nil *Table")
	}
}

// TestNextLightUserdataKey verifies that lightuserdata values (e.g. from
// debug.upvalueid) can be used as table keys and are visible via Next().
func TestNextLightUserdataKey(t *testing.T) {
	uv := NewOpenUpvalue(nil, 0)
	key := NewUpvalueID(uv)

	tbl := NewEmptyTable()
	tbl.MustSet(key, NewString("ok"))

	// Verify Get works
	got := tbl.Get(key)
	if got.AsString() != "ok" {
		t.Fatalf("Get(upvalueID) = %v, want 'ok'", got)
	}

	// Verify Next finds the key
	k, v, err := tbl.Next(Nil)
	if err != nil {
		t.Fatalf("Next(nil) error: %v", err)
	}
	if k.IsNil() {
		t.Fatal("Next(nil) returned nil key; lightuserdata key should be visible")
	}
	if k.Type() != "userdata" {
		t.Errorf("Next key type = %q, want 'userdata'", k.Type())
	}
	if !k.Equal(key) {
		t.Errorf("Next key not equal to original upvalueID")
	}
	if v.AsString() != "ok" {
		t.Errorf("Next value = %v, want 'ok'", v)
	}
}
