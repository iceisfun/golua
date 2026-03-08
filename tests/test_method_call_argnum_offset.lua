-- When string methods are called via method syntax (s:format(...)),
-- error argument numbers should be relative to the visible arguments
-- (not counting the implicit self), and function name should be short.
--
-- Lua 5.4: s:format("hello") -> "bad argument #1 to 'format'"
-- GoLua:  s:format("hello") -> "bad argument #2 to 'string.format'" (WRONG)
local ok, err

-- s:format with bad arg: should be arg #1, name 'format'
ok, err = pcall(function()
  local s = "%d"
  return s:format("hello")
end)
assert(err:find("#1"), "expected arg #1, got: " .. tostring(err))
assert(err:find("'format'") and not err:find("'string.format'"),
  "expected 'format', got: " .. tostring(err))

-- s:gsub with bad replacement: should be arg #2, name 'gsub'
ok, err = pcall(function()
  local s = "hello"
  return s:gsub("l", true)
end)
assert(err:find("#2"), "expected arg #2, got: " .. tostring(err))
assert(err:find("'gsub'") and not err:find("'string.gsub'"),
  "expected 'gsub', got: " .. tostring(err))

print("OK")
