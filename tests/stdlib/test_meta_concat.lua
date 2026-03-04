-- Test: events.lua - __concat metamethod
-- From: events.lua
-- What: Tests string concatenation metamethod with chained and mixed operands

do
  local t = {}
  t.__concat = function (a,b,c)
    assert(c == nil)
    if type(a) == 'table' then a = a.val end
    if type(b) == 'table' then b = b.val end
    if A then return a..b
    else
      return setmetatable({val=a..b}, t)
    end
  end

  local c = {val="c"}; setmetatable(c, t)
  local d = {val="d"}; setmetatable(d, t)

  A = true
  assert(c..d == 'cd')
  assert(0 .."a".."b"..c..d.."e".."f"..(5+3).."g" == "0abcdef8g")

  A = false
  assert((c..d..c..d).val == 'cdcd')
  local x = c..d
  assert(getmetatable(x) == t and x.val == 'cd')
  x = 0 .."a".."b"..c..d.."e".."f".."g"
  assert(x.val == "0abcdefg")

  A = nil
end
