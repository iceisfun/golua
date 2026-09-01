package golua_test

import (
	"strconv"
	"strings"
	"testing"
)

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

// A bare global name is sugar for _ENV[name], so a global target conflicts
// with a later target in the same statement that rebinds _ENV itself: the
// stores run right-to-left, so _ENV would already hold the new environment by
// the time the global is stored. Reference Lua takes one safe copy of the
// environment at the point the rebinding target is parsed — after the code for
// every earlier target's operands, before the code for any later one — and
// routes every global target parsed since the previous copy through it.
//
// The expected values below were captured from /usr/bin/lua5.4.8.

func TestAssignOrder_GlobalTargetWithEnvRebind(t *testing.T) {
	src := `
		local assert, rawget, setmetatable = assert, rawget, setmetatable
		local old = _ENV
		local new = setmetatable({}, {__index = old})
		x, _ENV = 1, new
		assert(_ENV == new, "_ENV was not rebound")
		assert(rawget(old, "x") == 1, "old.x="..tostring(rawget(old, "x")))
		assert(rawget(new, "x") == nil, "new.x="..tostring(rawget(new, "x")))
	`
	runLuaSource(t, src, "assign_global_env_rebind")
}

func TestAssignOrder_EnvRebindBeforeGlobalTarget(t *testing.T) {
	// The _ENV target comes first, so its store happens last (right-to-left)
	// and no copy is needed: the global still lands in the original
	// environment because that is what _ENV still holds when it is stored.
	src := `
		local assert, rawget, setmetatable = assert, rawget, setmetatable
		local old = _ENV
		local new = setmetatable({}, {__index = old})
		_ENV, x = new, 1
		assert(_ENV == new, "_ENV was not rebound")
		assert(rawget(old, "x") == 1, "old.x="..tostring(rawget(old, "x")))
		assert(rawget(new, "x") == nil, "new.x="..tostring(rawget(new, "x")))
	`
	runLuaSource(t, src, "assign_env_rebind_before_global")
}

func TestAssignOrder_GlobalTargetsAroundEnvRebind(t *testing.T) {
	src := `
		local assert, rawget, setmetatable = assert, rawget, setmetatable
		local old = _ENV
		local new = setmetatable({}, {__index = old})
		a, _ENV, b = 1, new, 2
		assert(rawget(old, "a") == 1, "old.a="..tostring(rawget(old, "a")))
		assert(rawget(old, "b") == 2, "old.b="..tostring(rawget(old, "b")))
		assert(rawget(new, "a") == nil, "new.a="..tostring(rawget(new, "a")))
		assert(rawget(new, "b") == nil, "new.b="..tostring(rawget(new, "b")))
	`
	runLuaSource(t, src, "assign_globals_around_env_rebind")
}

func TestAssignOrder_TwoGlobalTargetsWithEnvRebind(t *testing.T) {
	src := `
		local assert, rawget, setmetatable = assert, rawget, setmetatable
		local old = _ENV
		local new = setmetatable({}, {__index = old})
		x, y, _ENV = 1, 2, new
		assert(rawget(old, "x") == 1 and rawget(old, "y") == 2,
			"globals did not land in the original environment")
		assert(rawget(new, "x") == nil and rawget(new, "y") == nil,
			"globals leaked into the new environment")
	`
	runLuaSource(t, src, "assign_two_globals_env_rebind")
}

func TestAssignOrder_GlobalTargetWithLocalEnvRebind(t *testing.T) {
	// Same conflict when _ENV is a local rather than an upvalue: the copy is
	// an OP_MOVE instead of an OP_GETUPVAL.
	src := `
		local assert, rawget, setmetatable = assert, rawget, setmetatable
		local outer = _ENV
		do
			local _ENV = outer
			local new = setmetatable({}, {__index = outer})
			x, _ENV = 1, new
			assert(_ENV == new, "_ENV was not rebound")
			assert(rawget(outer, "x") == 1, "outer.x="..tostring(rawget(outer, "x")))
			assert(rawget(new, "x") == nil, "new.x="..tostring(rawget(new, "x")))
		end
	`
	runLuaSource(t, src, "assign_global_local_env_rebind")
}

func TestAssignOrder_EnvParameterRebindWithFieldTarget(t *testing.T) {
	// _ENV as an explicit function parameter, with both an explicit _ENV.z
	// target and a bare global in the same statement.
	src := `
		local assert, rawget, setmetatable = assert, rawget, setmetatable
		local function f(_ENV, new)
			local old = _ENV
			_ENV.z, w, _ENV = 9, 8, new
			return rawget(old, "z"), rawget(old, "w"), rawget(new, "z"), rawget(new, "w"), _ENV
		end
		local base, fresh = {}, {}
		local oz, ow, nz, nw, e = f(base, fresh)
		assert(oz == 9, "base.z="..tostring(oz))
		assert(ow == 8, "base.w="..tostring(ow))
		assert(nz == nil, "fresh.z="..tostring(nz))
		assert(nw == nil, "fresh.w="..tostring(nw))
		assert(e == fresh, "_ENV was not rebound")
	`
	runLuaSource(t, src, "assign_env_param_rebind")
}

func TestAssignOrder_GlobalTargetWithEnvRebindMultiRet(t *testing.T) {
	// The values come from one multi-return call, so the copy must be taken
	// before the call runs.
	src := `
		local assert, rawget, setmetatable = assert, rawget, setmetatable
		local old = _ENV
		local new = setmetatable({}, {__index = old})
		local function two() return 1, new end
		x, _ENV = two()
		assert(_ENV == new, "_ENV was not rebound")
		assert(rawget(old, "x") == 1, "old.x="..tostring(rawget(old, "x")))
		assert(rawget(new, "x") == nil, "new.x="..tostring(rawget(new, "x")))
	`
	runLuaSource(t, src, "assign_global_env_rebind_multiret")
}

func TestAssignOrder_GlobalTargetEnvRebindByRHS(t *testing.T) {
	// No target rebinds _ENV, so there is no conflict and no copy: the store
	// uses the live environment, which the RHS has already swapped.
	src := `
		local assert, rawget, setmetatable = assert, rawget, setmetatable
		local old = _ENV
		local new = setmetatable({}, {__index = old})
		local function swap() _ENV = new; return 7 end
		local z
		x, z = swap(), 1
		assert(rawget(old, "x") == nil, "old.x="..tostring(rawget(old, "x")))
		assert(rawget(new, "x") == 7, "new.x="..tostring(rawget(new, "x")))
	`
	runLuaSource(t, src, "assign_global_env_rebind_by_rhs")
}

func TestAssignOrder_EnvRebindAfterSideEffectingKey(t *testing.T) {
	// The copy is emitted where the _ENV target is parsed, so it runs AFTER
	// the key operand of the intervening target. f() has already installed
	// the new environment by then, so the global lands in the new table.
	src := `
		local assert, rawget, setmetatable = assert, rawget, setmetatable
		local old = _ENV
		local new = setmetatable({}, {__index = old})
		local t = {}
		local function f() _ENV = new; return "k" end
		x, t[f()], _ENV = 1, 2, new
		assert(t.k == 2, "t.k="..tostring(t.k))
		assert(_ENV == new, "_ENV was not rebound")
		assert(rawget(old, "x") == nil, "old.x="..tostring(rawget(old, "x")))
		assert(rawget(new, "x") == 1, "new.x="..tostring(rawget(new, "x")))
	`
	runLuaSource(t, src, "assign_env_rebind_after_key_side_effect")
}

func TestAssignOrder_EnvRebindAfterSideEffectingTable(t *testing.T) {
	// Same, with the side effect in the intervening target's table operand.
	src := `
		local assert, rawget, setmetatable = assert, rawget, setmetatable
		local old = _ENV
		local new = setmetatable({}, {__index = old})
		local function f() _ENV = new; return {} end
		x, (f()).k, _ENV = 1, 2, new
		assert(_ENV == new, "_ENV was not rebound")
		assert(rawget(old, "x") == nil, "old.x="..tostring(rawget(old, "x")))
		assert(rawget(new, "x") == 1, "new.x="..tostring(rawget(new, "x")))
	`
	runLuaSource(t, src, "assign_env_rebind_after_table_side_effect")
}

func TestAssignOrder_SideEffectingTargetAfterEnvRebind(t *testing.T) {
	// Mirror image: the side effect is in a target parsed AFTER the _ENV
	// target, so it runs after the copy and cannot affect the global's store.
	src := `
		local assert, rawget, setmetatable = assert, rawget, setmetatable
		local old = _ENV
		local new = setmetatable({}, {__index = old})
		local other = setmetatable({}, {__index = old})
		local t = {}
		local function g() _ENV = other; return "k" end
		x, _ENV, t[g()] = 1, new, 2
		assert(t.k == 2, "t.k="..tostring(t.k))
		assert(_ENV == new, "the _ENV target stores last, after g() ran")
		assert(rawget(old, "x") == 1, "old.x="..tostring(rawget(old, "x")))
		assert(rawget(new, "x") == nil, "new.x="..tostring(rawget(new, "x")))
		assert(rawget(other, "x") == nil, "other.x="..tostring(rawget(other, "x")))
	`
	runLuaSource(t, src, "assign_side_effect_after_env_rebind")
}

func TestAssignOrder_RepeatedEnvRebindWithGlobals(t *testing.T) {
	// Two rebinding targets: each takes its own copy, and every global is
	// routed through the copy taken by the next rebind to its right.
	src := `
		local assert, rawget, setmetatable = assert, rawget, setmetatable
		local old = _ENV
		local n1 = setmetatable({}, {__index = old})
		local n2 = setmetatable({}, {__index = old})
		x, _ENV, y, _ENV = 1, n1, 2, n2
		assert(_ENV == n1, "the leftmost _ENV store wins")
		assert(rawget(old, "x") == 1, "old.x="..tostring(rawget(old, "x")))
		assert(rawget(old, "y") == 2, "old.y="..tostring(rawget(old, "y")))
		assert(rawget(n1, "x") == nil and rawget(n1, "y") == nil, "globals leaked into n1")
		assert(rawget(n2, "x") == nil and rawget(n2, "y") == nil, "globals leaked into n2")
	`
	runLuaSource(t, src, "assign_repeated_env_rebind")
}

func TestAssignOrder_EnvRebindInLoop(t *testing.T) {
	// The copy is re-taken on every iteration, so each global lands in the
	// environment installed by the previous one.
	src := `
		local assert, rawget, setmetatable = assert, rawget, setmetatable
		local old = _ENV
		local envs = {}
		for i = 1, 3 do envs[i] = setmetatable({}, {__index = old}) end
		for i = 1, 3 do
			n, _ENV = i, envs[i]
		end
		assert(rawget(old, "n") == 1, "old.n="..tostring(rawget(old, "n")))
		assert(rawget(envs[1], "n") == 2, "envs[1].n="..tostring(rawget(envs[1], "n")))
		assert(rawget(envs[2], "n") == 3, "envs[2].n="..tostring(rawget(envs[2], "n")))
		assert(rawget(envs[3], "n") == nil, "envs[3].n="..tostring(rawget(envs[3], "n")))
	`
	runLuaSource(t, src, "assign_env_rebind_in_loop")
}

func TestAssignOrder_NestedFunctionEnvRebind(t *testing.T) {
	// _ENV reached as an upvalue several functions down; the chunk shares that
	// upvalue, so the rebind is visible in the main chunk too.
	src := `
		local assert, rawget, setmetatable = assert, rawget, setmetatable
		local old = _ENV
		local new = setmetatable({}, {__index = old})
		local function outer()
			local function middle()
				local function inner()
					x, _ENV = 1, new
					return _ENV
				end
				return inner()
			end
			return middle()
		end
		local e = outer()
		assert(e == new, "_ENV was not rebound in the innermost function")
		assert(_ENV == new, "the chunk shares that _ENV upvalue, so it is rebound too")
		assert(rawget(old, "x") == 1, "old.x="..tostring(rawget(old, "x")))
		assert(rawget(new, "x") == nil, "new.x="..tostring(rawget(new, "x")))
	`
	runLuaSource(t, src, "assign_nested_env_rebind")
}

func TestAssignOrder_GlobalTargetWithNonTableEnv(t *testing.T) {
	// The copy is what the store indexes, so a non-table environment fails
	// there instead of silently storing into whatever _ENV was rebound to.
	src := `
		local assert, pcall, load, tostring = assert, pcall, load, tostring
		local ok, err = pcall(load("local _ENV = 5; x, _ENV = 1, {}", "@e.lua"))
		assert(ok == false, "expected the store through the numeric environment to fail")
		assert(err == "e.lua:1: attempt to index a number value (local '_ENV')", "err="..tostring(err))

		local ok2, err2 = pcall(load("x, _ENV = 1, {}", "@u.lua", "t", 5))
		assert(ok2 == false, "expected the store through the numeric environment to fail")
		assert(err2 == "u.lua:1: attempt to index a number value (upvalue '_ENV')", "err2="..tostring(err2))
	`
	runLuaSource(t, src, "assign_global_non_table_env")
}

func TestAssignOrder_FieldTargetWithEnvRebind(t *testing.T) {
	// An ordinary field target is unaffected by an _ENV rebind, and the plain
	// local-table conflict keeps working alongside it.
	src := `
		local assert, rawget, setmetatable = assert, rawget, setmetatable
		local old = _ENV
		local new = setmetatable({}, {__index = old})
		local t = {}
		t.k, _ENV = 1, new
		assert(_ENV == new, "_ENV was not rebound")
		assert(t.k == 1, "t.k="..tostring(t.k))

		local a = {}
		local orig = a
		a.x, a = 1, {}
		assert(a ~= orig, "a was not rebound")
		assert(rawget(orig, "x") == 1, "orig.x="..tostring(rawget(orig, "x")))
		assert(rawget(a, "x") == nil, "the store leaked into the new table")
	`
	runLuaSource(t, src, "assign_field_target_env_rebind")
}

func TestAssignOrder_ShadowingLocalEnvNotATarget(t *testing.T) {
	// A local named _ENV that no target rebinds needs no copy: the global
	// lands in it, the way an ordinary local environment always does.
	src := `
		local assert, rawget, setmetatable = assert, rawget, setmetatable
		local outer = _ENV
		do
			local _ENV = setmetatable({}, {__index = outer})
			local mine = _ENV
			local other = {}
			x, other.k = 1, 2
			assert(rawget(mine, "x") == 1, "mine.x="..tostring(rawget(mine, "x")))
			assert(rawget(outer, "x") == nil, "outer.x="..tostring(rawget(outer, "x")))
			assert(other.k == 2, "other.k="..tostring(other.k))
		end
	`
	runLuaSource(t, src, "assign_shadowing_local_env")
}

// Reference Lua reserves exactly one register for the safe copy, shared by
// every global target to the left of the rebind. One copy per global target
// would exhaust the register file on a statement this wide.
func TestAssignOrder_ManyGlobalTargetsWithEnvRebind(t *testing.T) {
	const n = 160

	var targets, values strings.Builder
	for i := 0; i < n; i++ {
		targets.WriteString("g")
		targets.WriteString(strconv.Itoa(i))
		targets.WriteString(", ")
		values.WriteString(strconv.Itoa(i))
		values.WriteString(", ")
	}

	src := `
		local assert, rawget, setmetatable = assert, rawget, setmetatable
		local old = _ENV
		local new = setmetatable({}, {__index = old})
		` + targets.String() + `_ENV = ` + values.String() + `new
		assert(_ENV == new, "_ENV was not rebound")
		for i = 0, ` + strconv.Itoa(n-1) + ` do
			local name = "g"..i
			assert(rawget(old, name) == i, name.."="..tostring(rawget(old, name)))
			assert(rawget(new, name) == nil, "new."..name.." was set")
		end
	`
	runLuaSource(t, src, "assign_many_globals_env_rebind")
}
