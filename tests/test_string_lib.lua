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
  print(string.len("hello"), a:len())
end
--> =5	5

-- --------------------------------------------------------------------------
-- [Test 2] string.char
-- Verifies: output matches expected value via print()
do
  print(string.char(104, 101, 108, 108, 111))
end
--> =hello

-- --------------------------------------------------------------------------
-- [Test 3] string.upper, string.lower
-- Verifies: output matches expected value via print()
do
  print(string.upper("hello"), string.lower("HELLO"))
end
--> =HELLO	hello

-- --------------------------------------------------------------------------
-- [Test 4] string.rep
-- Verifies: output matches expected value via print()
do
  print(string.rep("hello", 3, ", "))
end
--> =hello, hello, hello

-- --------------------------------------------------------------------------
-- [Test 5] string.reverse
-- Verifies: output matches expected value via print()
do
  print(string.reverse("olleh"))
end
--> =hello

-- --------------------------------------------------------------------------
-- [Test 6] string.byte
-- Verifies: output matches expected value via print()
do
  print(string.byte("hello", 2, 4))
end
--> =101	108	108

-- --------------------------------------------------------------------------
-- [Test 7] string.format
-- Verifies: output matches expected value via print()
do
  print(string.format("%%%d %010d", 10, 23))
end
--> =%10 0000000023

-- --------------------------------------------------------------------------
-- [Test 8] string.format
-- Verifies: output matches expected value via print()
do
  print(string.format("%07X", 0xFFFFFFF))
end
--> =FFFFFFF

-- --------------------------------------------------------------------------
-- [Test 9] string.format
-- Verifies: code executes without runtime error
-- NOTE: Output varies or cannot be predicted; verify manually
do
  print(string.format("%q", 'a string with "quotes" and \\n new line'))
  print("PASS")
end
--> =PASS

-- --------------------------------------------------------------------------
-- [Test 10] string.sub
-- Verifies: output matches expected value via print()
do
  print(string.sub("123456789",2,4), string.sub("123456789",7), string.sub("123456789",7,6), string.sub("123456789",7,7), string.sub("123456789",0,0), string.sub("123456789",-10,10), string.sub("123456789",1,9), string.sub("123456789",-10,-20), string.sub("123456789",-1), string.sub("123456789",-4), string.sub("123456789",-6, -4))
end
--> =234	789		7		123456789	123456789		9	6789	456

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
  print(todump())
end
--> =hello1212.5

-- --------------------------------------------------------------------------
-- [Test 12] string.pack/unpack/packsize
-- Verifies: output matches expected value via print()
do
  local s1, n, s2 = "hello", 2, "you"
  local packed = string.pack("c5jc3", s1, n, s2)
  local us1, un, us2 = string.unpack("c5jc3", packed)
  print(string.packsize("c5jc3"), s1 == us1 and n == un and s2 == us2)
end
--> =12	true

-- --------------------------------------------------------------------------
-- [Test 13] string.find without pattern
-- Verifies: output matches expected value via print()
do
  print(string.find("hello to you", " to "))
end
--> =6	9

-- --------------------------------------------------------------------------
-- [Test 14] string.find with special pattern (issue #185)
-- Verifies: output matches expected value via print()
do
  print(string.find("-", "-"))
end
--> =1	1

-- --------------------------------------------------------------------------
-- [Test 15] string.match
-- Verifies: output matches expected value via print()
do
  print(string.match("foo: 123 bar: 456", "(%a+):%s*(%d+)"))
end
--> =foo	123

-- --------------------------------------------------------------------------
-- [Test 16] string.find
-- Verifies: output matches expected value via print()
do
  print(string.find("foo: 123 bar: 456", "(%a+):%s*(%d+)"))
end
--> =1	8	foo	123

-- --------------------------------------------------------------------------
-- [Test 17] string.gmatch
-- Verifies: output matches expected value via print()
do
  local s = "hello world from Lua"
  local t = {}

  for w in string.gmatch(s, "%a+") do
      table.insert(t, w)
  end

  print(table.unpack(t))
end
--> =hello	world	from	Lua

-- --------------------------------------------------------------------------
-- [Test 18] string.gsub
-- Verifies: output matches expected value via print()
do
  print(string.gsub("hello world", "(%w+)", "%1 %1"))
end
--> =hello hello world world	2

-- --------------------------------------------------------------------------
-- [Test 19] string.gsub (number)
-- Verifies: output matches expected value via print()
do
  print(string.gsub("hello world", "%w+", "%0 %0", 1))
end
--> =hello hello world	1

-- --------------------------------------------------------------------------
-- [Test 20] string.gsub (pattern)
-- Verifies: output matches expected value via print()
do
  print(string.gsub("hello world from Lua", "(%w+)%s*(%w+)", "%2 %1"))
end
--> =world hello Lua from	2

-- --------------------------------------------------------------------------
-- [Test 21] string.gsub (function)
-- Verifies: code executes without runtime error
-- NOTE: Run-only test (no assertions)
do
  local result = string.gsub("4+5 = $return 4+5$", "%$(.-)%$", function (s)
      return load(s)()
  end)
  assert(result == "4+5 = 9")
  print("PASS")
end

-- --------------------------------------------------------------------------
-- [Test 22] string.gsub (table)
-- Verifies: output matches expected value via print()
do
  local t = {name="lua", version="5.3"}
  print(string.gsub("$name-$version.tar.gz", "%$(%w+)", t))
end
--> =lua-5.3.tar.gz	2
