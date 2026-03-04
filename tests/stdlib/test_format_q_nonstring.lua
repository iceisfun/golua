-- Test: string.format %q for non-string types
-- From: strings.lua
-- What: Tests string.format("%q", ...) with integers, floats (including special values like inf, NaN, huge), booleans, and nil, verifying round-trip via load.

do
local function checkerror (msg, f, ...)
  local s, err = pcall(f, ...)
  assert(not s and string.find(err, msg))
end

do
  local function checkQ (v)
    local s = string.format("%q", v)
    local nv = load("return " .. s)()
    assert(v == nv and math.type(v) == math.type(nv))
  end
  checkQ("\0\0\1\255\u{234}")
  checkQ(math.maxinteger)
  checkQ(math.mininteger)
  checkQ(math.pi)
  checkQ(0.1)
  checkQ(true)
  checkQ(nil)
  checkQ(false)
  checkQ(math.huge)
  checkQ(-math.huge)
  assert(string.format("%q", 0/0) == "(0/0)")   -- NaN
  checkerror("no literal", string.format, "%q", {})
end
end
