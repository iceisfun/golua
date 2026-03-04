-- Test: pm.lua - Recursive gsub (string reversal)
-- From: pm.lua
-- What: Tests recursive calls to string.gsub to reverse a string, verifying that rev(rev(x)) == x.

do
  local function rev (s)
    return string.gsub(s, "(.)(.+)", function (c,s1) return rev(s1)..c end)
  end

  local x = "abcdef"
  assert(rev(rev(x)) == x)
end
