-- Test: Reuse of long strings (string interning)
-- From: literals.lua
-- What: Tests that identical long string constants are interned (share the same memory address), using string.format("%p") to compare pointer addresses.

do  -- reuse of long strings

  -- get the address of a string
  local function getadd (s) return string.format("%p", s) end

  local s1 <const> = "01234567890123456789012345678901234567890123456789"
  local s2 <const> = "01234567890123456789012345678901234567890123456789"
  local s3 = "01234567890123456789012345678901234567890123456789"
  local function foo() return s1 end
  local function foo1() return s3 end
  local function foo2()
    return "01234567890123456789012345678901234567890123456789"
  end
  local a1 = getadd(s1)
  assert(a1 == getadd(s2))
  assert(a1 == getadd(foo()))
  assert(a1 == getadd(foo1()))
  assert(a1 == getadd(foo2()))

  local sd = "0123456789" .. "0123456789012345678901234567890123456789"
  assert(sd == s1 and getadd(sd) ~= a1)
end
