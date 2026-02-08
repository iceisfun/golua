package vm

import "testing"

// TestTableChurnSetDelete exercises rapid Set/Delete cycles ("hole punching")
// and verifies that Get always returns a valid Nil for deleted keys, never
// a raw Go nil or stale data.
func TestTableChurnSetDelete(t *testing.T) {
	tbl := NewEmptyTable()

	for i := int64(1); i <= 10000; i++ {
		tbl.Set(NewInt(i), NewInt(i))
		if i%2 == 0 {
			tbl.Set(NewInt(i), Nil)
		}
	}

	// Verify: odd keys present, even keys return Nil
	for i := int64(1); i <= 10000; i++ {
		v := tbl.Get(NewInt(i))
		if i%2 == 0 {
			if !v.IsNil() {
				t.Fatalf("key %d should be nil after deletion, got %s", i, v.String())
			}
		} else {
			if v.IsNil() {
				t.Fatalf("key %d should be %d, got nil", i, i)
			}
			if v.AsInt() != i {
				t.Fatalf("key %d should be %d, got %s", i, i, v.String())
			}
		}
	}
}

// TestTableChurnGetNeverReturnsRawNil verifies that Get on a missing key
// returns the canonical Nil value, not a zero Value that might differ.
func TestTableChurnGetNeverReturnsRawNil(t *testing.T) {
	tbl := NewEmptyTable()

	// Unset key
	v := tbl.Get(NewString("nonexistent"))
	if !v.IsNil() {
		t.Fatal("Get on missing key should return Nil")
	}

	// Set and delete via hash
	tbl.Set(NewString("key"), NewString("value"))
	tbl.Set(NewString("key"), Nil)
	v = tbl.Get(NewString("key"))
	if !v.IsNil() {
		t.Fatalf("Get on deleted hash key should return Nil, got %s", v.String())
	}

	// Set and delete via array
	tbl.Set(NewInt(1), NewInt(42))
	tbl.Set(NewInt(1), Nil)
	v = tbl.Get(NewInt(1))
	if !v.IsNil() {
		t.Fatalf("Get on deleted array key should return Nil, got %s", v.String())
	}
}

// TestTableChurnNoPanic verifies that no panic occurs during rapid
// insert/delete across both array and hash parts.
func TestTableChurnNoPanic(t *testing.T) {
	tbl := NewEmptyTable()

	// Mixed operations: sequential insert, random-ish delete, re-insert
	for round := 0; round < 3; round++ {
		for i := int64(1); i <= 1000; i++ {
			tbl.Set(NewInt(i), NewInt(i*10))
		}
		for i := int64(1); i <= 1000; i += 3 {
			tbl.Set(NewInt(i), Nil)
		}
		for i := int64(1); i <= 1000; i += 3 {
			tbl.Set(NewInt(i), NewString("refilled"))
		}
	}

	// String keys
	for i := 0; i < 1000; i++ {
		key := NewString("k" + string(rune('A'+i%26)))
		tbl.Set(key, NewInt(int64(i)))
		if i%2 == 0 {
			tbl.Set(key, Nil)
		}
	}
}

// TestTableNextSkipsArrayHoles verifies that Next() skips nil-valued
// array entries created by deletion.
func TestTableNextSkipsArrayHoles(t *testing.T) {
	tbl := NewEmptyTable()

	// Create array [1, 2, 3, 4, 5]
	for i := int64(1); i <= 5; i++ {
		tbl.Set(NewInt(i), NewInt(i*10))
	}

	// Punch holes: delete keys 2 and 4
	tbl.Set(NewInt(2), Nil)
	tbl.Set(NewInt(4), Nil)

	// Iterate and collect keys
	var keys []int64
	k, v := tbl.Next(Nil)
	for !k.IsNil() {
		if v.IsNil() {
			t.Fatalf("Next() returned non-nil key %s with nil value", k.String())
		}
		ki, ok := k.ToInt()
		if !ok {
			t.Fatalf("expected integer key, got %s", k.String())
		}
		keys = append(keys, ki)
		k, v = tbl.Next(k)
	}

	// Should see keys 1, 3, 5 (holes at 2 and 4 skipped)
	expected := []int64{1, 3, 5}
	if len(keys) != len(expected) {
		t.Fatalf("expected keys %v, got %v", expected, keys)
	}
	for i, ek := range expected {
		if keys[i] != ek {
			t.Fatalf("expected keys %v, got %v", expected, keys)
		}
	}
}

// TestTableChurnHashOnly exercises hash-only churn (keys beyond array range).
func TestTableChurnHashOnly(t *testing.T) {
	tbl := NewEmptyTable()

	// Use large integer keys that go straight to hash
	for i := int64(1000); i <= 2000; i++ {
		tbl.Set(NewInt(i), NewInt(i))
	}
	for i := int64(1000); i <= 2000; i += 2 {
		tbl.Set(NewInt(i), Nil)
	}

	// Verify
	for i := int64(1000); i <= 2000; i++ {
		v := tbl.Get(NewInt(i))
		if i%2 == 0 {
			if !v.IsNil() {
				t.Fatalf("hash key %d should be nil after deletion", i)
			}
		} else {
			if v.IsNil() || v.AsInt() != i {
				t.Fatalf("hash key %d should be %d", i, i)
			}
		}
	}
}
