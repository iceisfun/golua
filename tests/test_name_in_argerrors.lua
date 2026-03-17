-- __name metafield in argument type errors (luaL_typeerror equivalent)
-- Lua 5.4 uses __name for "got TYPE" in bad-argument errors across all
-- standard library functions, not just VM-level operation errors.

local mt = {__name = "Custom"}
local t = setmetatable({}, mt)

-- String library functions should report "got Custom"
local ok, err

ok, err = pcall(string.byte, t)
assert(err:find("got Custom"), "string.byte: " .. err)

ok, err = pcall(string.find, t, "x")
assert(err:find("got Custom"), "string.find: " .. err)

ok, err = pcall(string.len, t)
assert(err:find("got Custom"), "string.len: " .. err)

ok, err = pcall(string.lower, t)
assert(err:find("got Custom"), "string.lower: " .. err)

ok, err = pcall(string.upper, t)
assert(err:find("got Custom"), "string.upper: " .. err)

ok, err = pcall(string.sub, t, 1)
assert(err:find("got Custom"), "string.sub: " .. err)

ok, err = pcall(string.rep, t, 1)
assert(err:find("got Custom"), "string.rep: " .. err)

-- string.format integer specifiers should report "got Custom"
ok, err = pcall(string.format, "%d", t)
assert(err:find("got Custom"), "string.format %%d: " .. err)

ok, err = pcall(string.format, "%x", t)
assert(err:find("got Custom"), "string.format %%x: " .. err)

ok, err = pcall(string.format, "%f", t)
assert(err:find("got Custom"), "string.format %%f: " .. err)

-- string.pack should report "got Custom"
ok, err = pcall(string.pack, "i4", t)
assert(err:find("got Custom"), "string.pack: " .. err)

-- table.sort with non-function comparator should report "got Custom"
ok, err = pcall(table.sort, {1,2}, t)
assert(err:find("got Custom"), "table.sort: " .. err)

-- tonumber with base should report "got Custom"
ok, err = pcall(tonumber, t, 10)
assert(err:find("got Custom"), "tonumber: " .. err)

-- Global functions via gotDesc should report "got Custom"
ok, err = pcall(next, t)
-- next actually accepts tables, so this should succeed
assert(ok, "next should accept a table with __name")

print("OK")
