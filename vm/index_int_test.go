package vm

import "testing"

// TestIndexIntMatchesIndexValue verifies that IndexInt (the integer-key fast
// path used by stdlib table functions and the ipairs iterator) produces the
// same result as IndexValue with a boxed integer key across the frame-free code
// paths: plain array-backed table, integer-hash key, holes, a __index table
// chain, and a non-table value. The __index *function* metamethod path needs an
// active call frame and is exercised through Lua by
// TestProtectedCallTableUnpackWithMetamethods (which routes through IndexInt via
// tableGetIdx).
func TestIndexIntMatchesIndexValue(t *testing.T) {
	v := New()

	check := func(name string, obj Value, key int) {
		t.Helper()
		want, wErr := v.IndexValue(obj, NewInt(int64(key)))
		got, gErr := v.IndexInt(obj, key)
		if (wErr == nil) != (gErr == nil) {
			t.Fatalf("%s[%d]: error mismatch: IndexValue err=%v IndexInt err=%v", name, key, wErr, gErr)
		}
		// Value is not comparable with ==; compare by Lua content, and also
		// require the same type so an int/float coercion cannot hide a
		// divergence between the two lookup paths.
		if want.Type() != got.Type() || !want.RawEqual(got) {
			t.Fatalf("%s[%d]: IndexInt=%v (%s) but IndexValue=%v (%s)", name, key, got, got.Type(), want, want.Type())
		}
	}

	// 1. Plain table: array slot, beyond-array integer-hash slot, and a hole.
	plain := NewEmptyTable()
	plain.MustSet(NewInt(1), NewString("a"))
	plain.MustSet(NewInt(2), NewString("b"))
	plain.MustSet(NewInt(100), NewString("far")) // integer hash, outside array
	pv := NewTable(plain)
	check("plain", pv, 1)
	check("plain", pv, 2)
	check("plain", pv, 3)   // hole -> nil
	check("plain", pv, 100) // integer hash
	check("plain", pv, 0)   // non-positive -> nil

	// 2. __index table chain: missing keys fall through to a backing table.
	backing := NewEmptyTable()
	backing.MustSet(NewInt(42), NewString("from-backing"))
	withTbl := NewEmptyTable()
	withTbl.MustSet(NewInt(1), NewString("own"))
	mtTbl := NewEmptyTable()
	mtTbl.MustSet(NewString("__index"), NewTable(backing))
	withTbl.SetMetatable(mtTbl)
	tv := NewTable(withTbl)
	check("indexChain", tv, 1)  // own key
	check("indexChain", tv, 42) // via __index table
	check("indexChain", tv, 99) // absent in both -> nil

	// 3. Non-table value: indexing a number must error identically.
	if _, err := v.IndexInt(NewInt(3), 1); err == nil {
		t.Fatalf("IndexInt on a number should error")
	}
}
