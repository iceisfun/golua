package golua_test

import "testing"

// These tests pin golua's assignment evaluation order to reference Lua 5.4.8 /
// 5.5.0. For an indexed/field assignment `t[k] = e`, reference Lua references
// the *live* register of a bare-local table/key in the store instruction, so a
// reassignment of that local performed while evaluating the RHS `e` is observed
// by the store. golua previously snapshotted the table/key into a temp before
// the RHS, diverging from both reference versions.
//
// The expected values below were captured from /usr/bin/lua5.5.0 AND
// /usr/bin/lua (5.4.8) — they agree.

func TestAssignOrder_SingleIndexTableRebind(t *testing.T) {
	// f() rebinds local t to a fresh table; the store must land in that
	// fresh table (the live register), not the original.
	src := `
		local t = {0}
		local orig = t
		local function f() t = {9}; return 5 end
		t[1] = f()
		assert(orig[1] == 0, "orig[1]="..tostring(orig[1]))   -- original untouched
		assert(t ~= orig, "t was not rebound")
		assert(t[1] == 5, "t[1]="..tostring(t[1]))            -- store into new table
	`
	runLuaSource(t, src, "assign_single_index_table_rebind")
}

func TestAssignOrder_SingleIndexKeyMutated(t *testing.T) {
	// f() mutates the key local i; the store must use i's post-RHS value.
	src := `
		local t = {0, 0}
		local i = 1
		local function f() i = 2; return 99 end
		t[i] = f()
		assert(t[1] == 0, "t[1]="..tostring(t[1]))
		assert(t[2] == 99, "t[2]="..tostring(t[2]))
	`
	runLuaSource(t, src, "assign_single_index_key_mutated")
}

func TestAssignOrder_SingleFieldTableRebind(t *testing.T) {
	src := `
		local t = {x = 0}
		local orig = t
		local function f() t = {x = 7}; return 5 end
		t.x = f()
		assert(orig.x == 0, "orig.x="..tostring(orig.x))
		assert(t ~= orig, "t was not rebound")
		assert(t.x == 5, "t.x="..tostring(t.x))
	`
	runLuaSource(t, src, "assign_single_field_table_rebind")
}

func TestAssignOrder_SingleIndexSETI(t *testing.T) {
	// SETI fast path (constant integer key) with a rebound table local.
	src := `
		local t = {0}
		local orig = t
		local function f() t = {0}; return 42 end
		t[1] = f()
		assert(orig[1] == 0 and t[1] == 42 and t ~= orig)
	`
	runLuaSource(t, src, "assign_single_index_seti")
}

func TestAssignOrder_MultiIndexTableRebind(t *testing.T) {
	// Multi-assignment: RHS side effect rebinds the target's table local.
	src := `
		local t = {0}
		local orig = t
		local function f() t = {9}; return 5 end
		local a
		a, t[1] = 1, f()
		assert(a == 1)
		assert(orig[1] == 0, "orig[1]="..tostring(orig[1]))
		assert(t ~= orig, "t was not rebound")
		assert(t[1] == 5, "t[1]="..tostring(t[1]))
	`
	runLuaSource(t, src, "assign_multi_index_table_rebind")
}

// --- Conflict cases that must KEEP matching reference (later target reassigns
// a local used as an earlier indexed target's table/key -> snapshot via the
// equivalent of reference's check_conflict). ---

func TestAssignOrder_ConflictKeyLocal(t *testing.T) {
	// `a[k], k = "NEW", 2`: k is reassigned by a later target, so a[k] must
	// use k's ORIGINAL value (1).
	src := `
		local a = {[1]="x", [2]="y"}
		local k = 1
		a[k], k = "NEW", 2
		assert(a[1] == "NEW", "a[1]="..tostring(a[1]))
		assert(a[2] == "y",   "a[2]="..tostring(a[2]))
		assert(k == 2)
	`
	runLuaSource(t, src, "assign_conflict_key_local")
}

func TestAssignOrder_ConflictTableLocal(t *testing.T) {
	// `t[1], t = 5, u`: t is reassigned by a later target, so t[1]=5 must use
	// the ORIGINAL t (which is then discarded); u stays untouched.
	src := `
		local t = {0}
		local u = {0}
		t[1], t = 5, u
		assert(t == u)
		assert(u[1] == 0, "u[1]="..tostring(u[1]))
	`
	runLuaSource(t, src, "assign_conflict_table_local")
}
