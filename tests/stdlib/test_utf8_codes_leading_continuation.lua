-- utf8.codes should error on invalid UTF-8, including stray continuation bytes.
-- Lua 5.4 does NOT skip continuation bytes — it raises an error.

-- Leading continuation bytes → error
do
  local s = string.char(0xB2, 0xBD, 0xB8, 0x2A, 0x09, 0x5E)
  local ok, err = pcall(function()
    for p, c in utf8.codes(s) do end
  end)
  assert(not ok, "should error on leading continuation bytes")
  assert(string.find(tostring(err), "invalid UTF%-8 code"))
end

-- Stray continuation after valid codepoint → error
do
  local s = string.char(0x94, 0x7E, 0x69, 0xE5)
  local ok, err = pcall(function()
    for _p, _c in utf8.codes(s) do
    end
  end)
  assert(ok == false)
  assert(string.find(tostring(err), "invalid UTF-8 code", 1, true) ~= nil)
end

-- Trailing continuation byte after valid ASCII → error
do
  local s = string.char(0x61, 0xBA)
  local ok, err = pcall(function()
    for p, c in utf8.codes(s) do end
  end)
  assert(not ok, "should error on trailing continuation byte")
  assert(string.find(tostring(err), "invalid UTF%-8 code"))
end
