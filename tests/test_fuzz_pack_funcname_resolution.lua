-- test_fuzz_pack_funcname_resolution:
-- string.pack/unpack/packsize argument errors must resolve the function name
-- the way Lua 5.5 does: a DIRECT call resolves to the short field name
-- ('pack'), while a call made indirectly (via pcall) falls back to the
-- qualified global name ('string.pack'). golua used to hardcode the long
-- name for these internal panics.
--
-- Discovered: differential scout 2026-05-20 (string-pack agent, s1#2).

-- Direct call (callee resolved from the Lua caller's bytecode) -> short name.
local err = select(2, pcall(function() return string.pack("i1", 999) end))
assert(err:find("to 'pack'", 1, true),
  "direct string.pack call should report 'pack', got: " .. tostring(err))

local err2 = select(2, pcall(function() return string.unpack("i4", "ab") end))
assert(err2:find("to 'unpack'", 1, true),
  "direct string.unpack call should report 'unpack', got: " .. tostring(err2))

local err3 = select(2, pcall(function() return string.packsize("s4") end))
assert(err3:find("to 'packsize'", 1, true),
  "direct string.packsize call should report 'packsize', got: " .. tostring(err3))

-- Indirect call (callee passed to pcall) -> qualified name.
local err4 = select(2, pcall(string.pack, "i1", 999))
assert(err4:find("to 'string.pack'", 1, true),
  "pcall'd string.pack should report 'string.pack', got: " .. tostring(err4))

local err5 = select(2, pcall(string.unpack, "i4", "ab"))
assert(err5:find("to 'string.unpack'", 1, true),
  "pcall'd string.unpack should report 'string.unpack', got: " .. tostring(err5))

print("ok")
