-- Test: Const local variables
-- From: locals.lua
-- What: Tests the <const> attribute on local variables: that const variables cannot be reassigned, including through function definitions and nested closures.

do   -- constants
  local a<const>, b, c<const> = 10, 20, 30
  b = a + c + b    -- 'b' is not constant
  assert(a == 10 and b == 60 and c == 30)
  local function checkro (name, code)
    local st, msg = load(code)
    local gab = string.format("attempt to assign to const variable '%s'", name)
    assert(not st and string.find(msg, gab))
  end
  checkro("y", "local x, y <const>, z = 10, 20, 30; x = 11; y = 12")
  checkro("x", "local x <const>, y, z <const> = 10, 20, 30; x = 11")
  checkro("z", "local x <const>, y, z <const> = 10, 20, 30; y = 10; z = 11")
  checkro("foo", "local foo <const> = 10; function foo() end")
  checkro("foo", "local foo <const> = {}; function foo() end")

  checkro("z", [[
    local a, z <const>, b = 10;
    function foo() a = 20; z = 32; end
  ]])

  checkro("var1", [[
    local a, var1 <const> = 10;
    function foo() a = 20; z = function () var1 = 12; end  end
  ]])
end
