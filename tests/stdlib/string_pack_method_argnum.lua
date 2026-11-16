-- Test: value/format errors from string.pack/unpack/packsize apply the
-- method-call adjustment to the argument number like luaL_argerror does.
local function msg(f)
  local ok, err = pcall(f)
  assert(not ok)
  return (err:gsub("^[^:]+:%d+: ", ""))
end

assert(msg(function() return ("i4"):unpack("ab") end) ==
  "bad argument #1 to 'unpack' (data string too short)")
assert(msg(function() return ("i4"):unpack("abcd", 9) end) ==
  "bad argument #2 to 'unpack' (initial position out of string)")
assert(msg(function() return ("i1"):pack(300) end) ==
  "bad argument #1 to 'pack' (integer overflow)")
assert(msg(function() return ("s"):packsize() end) ==
  "calling 'packsize' on bad self (variable-length format)")

-- plain calls are unchanged
assert(msg(function() return string.unpack("i4", "ab") end) ==
  "bad argument #2 to 'unpack' (data string too short)")
assert(msg(function() return string.unpack("i4", "abcd", 9) end) ==
  "bad argument #3 to 'unpack' (initial position out of string)")
assert(msg(function() return string.pack("i1", 300) end) ==
  "bad argument #2 to 'pack' (integer overflow)")
assert(msg(function() return string.packsize("s") end) ==
  "bad argument #1 to 'packsize' (variable-length format)")
print("PASS")
