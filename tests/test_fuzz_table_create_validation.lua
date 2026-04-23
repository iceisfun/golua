-- Validates table.create(narr, nrec) argument checking (Lua 5.5 §6.6).
-- Hostile input (missing arg, negative size, implementation-exceeding size)
-- must produce the standard "bad argument" error WITHOUT leaking any
-- Go-internal runtime message (e.g. "makeslice: cap out of range").

local function expect_err(label, expected_substr, fn, ...)
    local ok, err = pcall(fn, ...)
    assert(not ok, label .. ": expected error, got success")
    assert(type(err) == "string", label .. ": expected string error, got " .. type(err))
    assert(err:find(expected_substr, 1, true),
        label .. ": expected error containing '" .. expected_substr ..
        "', got: " .. err)
    -- Guard against Go runtime internals leaking through the Lua error path.
    assert(not err:find("runtime error", 1, true),
        label .. ": Go runtime error leaked: " .. err)
    assert(not err:find("makeslice", 1, true),
        label .. ": Go makeslice error leaked: " .. err)
    assert(not err:find("cap out of range", 1, true),
        label .. ": Go cap-out-of-range leaked: " .. err)
end

-- Missing first argument: number expected, got no value.
expect_err("no args", "bad argument #1", table.create)
expect_err("no args msg", "number expected, got no value", table.create)

-- Negative narr: out of range.
expect_err("negative narr", "bad argument #1", table.create, -1)
expect_err("negative narr msg", "out of range", table.create, -1)

-- Negative nrec: out of range.
expect_err("negative nrec", "bad argument #2", table.create, 10, -1)
expect_err("negative nrec msg", "out of range", table.create, 10, -1)

-- Massive narr (beyond any reasonable preallocation): out of range.
expect_err("huge narr", "bad argument #1", table.create, 2^62)
expect_err("huge narr msg", "out of range", table.create, 2^62)

-- Massive nrec: out of range.
expect_err("huge nrec", "bad argument #2", table.create, 0, 2^62)
expect_err("huge nrec msg", "out of range", table.create, 0, 2^62)

-- Non-number first arg: number expected, got <type>.
expect_err("string narr", "bad argument #1", table.create, "abc")
expect_err("string narr msg", "number expected, got string", table.create, "abc")

-- Non-integer (float with fraction): "number has no integer representation".
expect_err("fractional narr", "bad argument #1", table.create, 1.5)
expect_err("fractional narr msg",
    "number has no integer representation", table.create, 1.5)

-- Happy path still works.
local t = table.create(4, 2)
assert(type(t) == "table")
assert(#t == 0)
t[1] = "x"
assert(t[1] == "x")
assert(#t == 1)

print("ok")
