-- Differential testing: string metatable and arithmetic metamethod priority
-- Lua 5.4: metamethods checked BEFORE string-to-number coercion

-- String metatable is separate from string library
do
  assert(getmetatable("") ~= string, "metatable should not be string table")
  assert(type(getmetatable("")) == "table")
  -- __index on string metatable points to string library
  assert(getmetatable("").__index == string)
end

-- String methods still work
do
  assert(("hello"):upper() == "HELLO")
  assert(("hello"):find("ell") == 2)
  assert(("hello"):len() == 5)
  assert(("hello"):sub(2, 4) == "ell")
  assert(string.len("hello") == 5)
end

-- Default arithmetic metamethods on string metatable do coercion
do
  assert("10" + 5 == 15)
  assert("10" - 5 == 5)
  assert("10" * 5 == 50)
  assert("10" / 5 == 2.0)
  assert("10" // 3 == 3)
  assert("10" % 3 == 1)
  assert("2" ^ 10 == 1024.0)
  assert(-"42" == -42)
end

-- Custom metamethods override coercion
do
  local mt = getmetatable("")
  local old_add = mt.__add

  mt.__add = function(a, b) return "custom_add" end
  assert("10" + 5 == "custom_add", "metamethod should override coercion")

  -- Restore
  mt.__add = old_add
  assert("10" + 5 == 15)
end

-- Unary minus metamethod override
do
  local mt = getmetatable("")
  local old_unm = mt.__unm
  mt.__unm = function(a) return "custom_unm" end
  assert(-"42" == "custom_unm")
  mt.__unm = old_unm
  assert(-"42" == -42)
end

print("PASS")
