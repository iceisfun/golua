-- Test: All 4-character combinations of bracket-related characters
-- From: literals.lua
-- What: Generates all 256 strings of length 4 using characters =, [, ], \n and verifies they survive round-tripping through a long bracket string with level 4.

do
  local x = {"=", "[", "]", "\n"}
  local len = 4
  local function gen (c, n)
    if n==0 then coroutine.yield(c)
    else
      for _, a in pairs(x) do
        gen(c..a, n-1)
      end
    end
  end

  for s in coroutine.wrap(function () gen("", len) end) do
    assert(s == load("return [====[\n"..s.."]====]", "")())
  end
end
