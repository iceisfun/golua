package golua_test

import "testing"

// TestTableCreateHugeHintNoFatalAbort is the regression guard for the bug where
// table.create with a large array/hash hint turned the advisory size hint into
// an eager multi-gigabyte Go slice allocation. make() zero-fills the backing
// store immediately, forcing every page resident, so the Go runtime aborted
// with a fatal (pcall-uncatchable) "out of memory" before pcall could see it.
//
// Reference Lua 5.5 succeeds on these because it relies on lazy page commit;
// golua clamps the real preallocation (the hint is advisory) so the table is
// returned just like reference, without the giant allocation.
func TestTableCreateHugeHintNoFatalAbort(t *testing.T) {
	runLuaSource(t, `
		-- A huge but in-range narr/nrec must SUCCEED (return a usable table),
		-- never fatally abort the process. 1<<29 is the originally reported value.
		local ok, t1 = pcall(table.create, 1 << 29)
		assert(ok, "table.create(1<<29) should succeed, got: " .. tostring(t1))
		assert(type(t1) == "table", "expected a table")

		-- INT_MAX (2^31-1) is the largest accepted array hint, matching reference.
		local ok2, t2 = pcall(table.create, (1 << 31) - 1)
		assert(ok2, "table.create(INT_MAX) should succeed, got: " .. tostring(t2))
		assert(type(t2) == "table", "expected a table")

		-- A huge hash hint up to MAXHSIZE (2^30) is accepted too.
		local ok3, t3 = pcall(table.create, 0, 1 << 30)
		assert(ok3, "table.create(0, 1<<30) should succeed, got: " .. tostring(t3))
		assert(type(t3) == "table", "expected a table")

		-- The returned table is fully functional despite the clamped prealloc.
		local t = table.create(1 << 29)
		for i = 1, 1000 do t[i] = i * 2 end
		assert(#t == 1000, "expected length 1000, got " .. tostring(#t))
		assert(t[500] == 1000, "expected t[500]==1000, got " .. tostring(t[500]))
	`, "table_create_huge_hint")
}

// TestTableCreateArgBoundaries pins the acceptance/error boundaries to match
// reference Lua 5.5: an argument above INT_MAX is the argument error
// "out of range" (with a source-location prefix), while a hash hint above
// MAXHSIZE is the plain runtime error "table overflow" (no location prefix).
func TestTableCreateArgBoundaries(t *testing.T) {
	runLuaSource(t, `
		local function fails(msg, ...)
			local ok, err = pcall(table.create, ...)
			assert(not ok, "expected failure but succeeded")
			assert(tostring(err):find(msg, 1, true),
				"expected error containing '" .. msg .. "', got: " .. tostring(err))
		end

		-- narr above INT_MAX -> out of range
		fails("bad argument #1 to 'table.create' (out of range)", 1 << 31)
		fails("bad argument #1 to 'table.create' (out of range)", 1 << 40)
		fails("bad argument #1 to 'table.create' (out of range)", -1)

		-- nrec above INT_MAX -> out of range (argument check fires first)
		fails("bad argument #2 to 'table.create' (out of range)", 0, 1 << 31)
		fails("bad argument #2 to 'table.create' (out of range)", 0, -1)

		-- nrec in (MAXHSIZE, INT_MAX] -> plain runtime "table overflow",
		-- WITHOUT a source-location prefix (distinct from the argument error).
		local ok, err = pcall(table.create, 0, (1 << 30) + 1)
		assert(not ok, "expected table overflow failure")
		assert(err == "table overflow",
			"expected bare 'table overflow' (no location prefix), got: " .. tostring(err))
	`, "table_create_boundaries")
}
