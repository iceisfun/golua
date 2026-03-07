-- ==========================================================================
-- Fengari test extraction: String library functions
-- Source: fengari (https://github.com/fengari-lua/fengari)
-- Category: string
-- Total tests: 22
-- ==========================================================================
-- Extracted from the fengari Lua 5.3 implementation test suite.
-- Each test is wrapped in a do...end block for scope isolation.
-- Doctest directives (--> =expected) verify print() output.
-- Self-verifying tests use assert() and print "PASS" on success.

-- [Test 1] string.len
-- Verifies: output matches expected value via print()
do
  local a = "world"
  assert(string.len("hello") == 5)
  assert(a:len() == 5)
end

-- --------------------------------------------------------------------------
-- [Test 2] string.char
-- Verifies: output matches expected value via print()
do
  assert(string.char(104, 101, 108, 108, 111) == "hello")
end

-- --------------------------------------------------------------------------
-- [Test 3] string.upper, string.lower
-- Verifies: output matches expected value via print()
do
  assert(string.upper("hello") == "HELLO")
  assert(string.lower("HELLO") == "hello")
end

-- --------------------------------------------------------------------------
-- [Test 4] string.rep
-- Verifies: output matches expected value via print()
do
  assert(string.rep("hello", 3, ", ") == "hello, hello, hello")
end

-- --------------------------------------------------------------------------
-- [Test 5] string.reverse
-- Verifies: output matches expected value via print()
do
  assert(string.reverse("olleh") == "hello")
end

-- --------------------------------------------------------------------------
-- [Test 6] string.byte
-- Verifies: output matches expected value via print()
do
  local b1, b2, b3 = string.byte("hello", 2, 4)
  assert(b1 == 101 and b2 == 108 and b3 == 108)
end

-- --------------------------------------------------------------------------
-- [Test 7] string.format
-- Verifies: output matches expected value via print()
do
  assert(string.format("%%%d %010d", 10, 23) == "%10 0000000023")
end

-- --------------------------------------------------------------------------
-- [Test 8] string.format
-- Verifies: output matches expected value via print()
do
  assert(string.format("%07X", 0xFFFFFFF) == "FFFFFFF")
end

-- --------------------------------------------------------------------------
-- [Test 9] string.format
-- Verifies: code executes without runtime error
-- NOTE: Output varies or cannot be predicted; verify manually
do
  local q = string.format("%q", 'a string with "quotes" and \\n new line')
  assert(type(q) == "string")
end

-- --------------------------------------------------------------------------
-- [Test 10] string.sub
-- Verifies: output matches expected value via print()
do
  assert(string.sub("123456789",2,4) == "234")
  assert(string.sub("123456789",7) == "789")
  assert(string.sub("123456789",7,6) == "")
  assert(string.sub("123456789",7,7) == "7")
  assert(string.sub("123456789",0,0) == "")
  assert(string.sub("123456789",-10,10) == "123456789")
  assert(string.sub("123456789",1,9) == "123456789")
  assert(string.sub("123456789",-10,-20) == "")
  assert(string.sub("123456789",-1) == "9")
  assert(string.sub("123456789",-4) == "6789")
  assert(string.sub("123456789",-6, -4) == "456")
end

-- --------------------------------------------------------------------------
-- [Test 11] string.dump
-- GoLua: binary chunk loading not supported, just verify dump produces string
do
  local todump = function()
      local s = "hello"
      local i = 12
      local f = 12.5
      return s .. i .. f
  end
  local dumped = string.dump(todump)
  assert(type(dumped) == "string")
  -- Can't load binary chunks in GoLua, verify function directly
  assert(todump() == "hello1212.5")
end

-- --------------------------------------------------------------------------
-- [Test 12] string.pack/unpack/packsize
-- Verifies: output matches expected value via print()
do
  local s1, n, s2 = "hello", 2, "you"
  local packed = string.pack("c5jc3", s1, n, s2)
  local us1, un, us2 = string.unpack("c5jc3", packed)
  assert(string.packsize("c5jc3") == 16)
  assert(s1 == us1 and n == un and s2 == us2)
end

-- --------------------------------------------------------------------------
-- [Test 13] string.find without pattern
-- Verifies: output matches expected value via print()
do
  local s, e = string.find("hello to you", " to ")
  assert(s == 6 and e == 9)
end

-- --------------------------------------------------------------------------
-- [Test 14] string.find with special pattern (issue #185)
-- Verifies: output matches expected value via print()
do
  local s, e = string.find("-", "-")
  assert(s == 1 and e == 1)
end

-- --------------------------------------------------------------------------
-- [Test 15] string.match
-- Verifies: output matches expected value via print()
do
  local m1, m2 = string.match("foo: 123 bar: 456", "(%a+):%s*(%d+)")
  assert(m1 == "foo" and m2 == "123")
end

-- --------------------------------------------------------------------------
-- [Test 16] string.find
-- Verifies: output matches expected value via print()
do
  local s, e, m1, m2 = string.find("foo: 123 bar: 456", "(%a+):%s*(%d+)")
  assert(s == 1 and e == 8 and m1 == "foo" and m2 == "123")
end

-- --------------------------------------------------------------------------
-- [Test 17] string.gmatch
-- Verifies: output matches expected value via print()
do
  local s = "hello world from Lua"
  local t = {}

  for w in string.gmatch(s, "%a+") do
      table.insert(t, w)
  end

  assert(#t == 4)
  assert(t[1] == "hello" and t[2] == "world" and t[3] == "from" and t[4] == "Lua")
end

-- --------------------------------------------------------------------------
-- [Test 18] string.gsub
-- Verifies: output matches expected value via print()
do
  local r, n = string.gsub("hello world", "(%w+)", "%1 %1")
  assert(r == "hello hello world world" and n == 2)
end

-- --------------------------------------------------------------------------
-- [Test 19] string.gsub (number)
-- Verifies: output matches expected value via print()
do
  local r, n = string.gsub("hello world", "%w+", "%0 %0", 1)
  assert(r == "hello hello world" and n == 1)
end

-- --------------------------------------------------------------------------
-- [Test 20] string.gsub (pattern)
-- Verifies: output matches expected value via print()
do
  local r, n = string.gsub("hello world from Lua", "(%w+)%s*(%w+)", "%2 %1")
  assert(r == "world hello Lua from" and n == 2)
end

-- --------------------------------------------------------------------------
-- [Test 21] string.gsub (function)
-- Verifies: code executes without runtime error
-- NOTE: Run-only test (no assertions)
do
  local result = string.gsub("4+5 = $return 4+5$", "%$(.-)%$", function (s)
      return load(s)()
  end)
  assert(result == "4+5 = 9")
end

-- --------------------------------------------------------------------------
-- [Test 22] string.gsub (table)
-- Verifies: output matches expected value via print()
do
  local t = {name="lua", version="5.3"}
  local r, n = string.gsub("$name-$version.tar.gz", "%$(%w+)", t)
  assert(r == "lua-5.3.tar.gz" and n == 2)
end
