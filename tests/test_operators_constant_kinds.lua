-- Test: constructs.lua - Operators with different constant kinds
-- From: constructs.lua
-- What: Tests all arithmetic/bitwise/comparison operators with different kinds of operands (small int, large int, float)

do
  local operand = {3, 100, 5.0, -10, -5.0, 10000, -10000}
  local operator = {"+", "-", "*", "/", "//", "%", "^",
                    "&", "|", "^", "<<", ">>",
                    "==", "~=", "<", ">", "<=", ">=",}
  for _, op in ipairs(operator) do
    local f = assert(load(string.format([[return function (x,y)
                  return x %s y
                end]], op)))();
    for _, o1 in ipairs(operand) do
      for _, o2 in ipairs(operand) do
        local gab = f(o1, o2)
        _ENV.XX = o1
        local code = string.format("return XX %s %s", op, o2)
        local res = assert(load(code))()
        assert(res == gab)
      end
    end
  end
end
