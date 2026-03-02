-- Test: bwcoercion.lua - String metamethods for bitwise ops
-- From: bwcoercion.lua
-- What: Helper file that sets up string metatable metamethods (__band, __bor, __bxor, __shl, __shr, __bnot) to enable string-to-integer coercion for bitwise operations

do
  local function checkargs (x, y, name)
    if type(x) == "string" then x = tonumber(x) end
    if type(y) == "string" then y = tonumber(y) end
    if not x then error("bad argument #1 to '" .. name .. "' (number expected, got string)", 3) end
    if not y then error("bad argument #2 to '" .. name .. "' (number expected, got string)", 3) end
    return x, y
  end

  local smt = getmetatable("")
  smt.__band = function (x, y)
    local x, y = checkargs(x, y, "__band")
    return y and x & y or x
  end
  smt.__bor = function (x, y)
    local x, y = checkargs(x, y, "__bor")
    return y and x | y or x
  end
  smt.__bxor = function (x, y)
    local x, y = checkargs(x, y, "__bxor")
    return y and x ~ y or x
  end
  smt.__shl = function (x, y)
    local x, y = checkargs(x, y, "__shl")
    return y and x << y or x
  end
  smt.__shr = function (x, y)
    local x, y = checkargs(x, y, "__shr")
    return y and x >> y or x
  end
  smt.__bnot = function (x)
    local x, y = checkargs(x, x, "__bnot")
    return y and ~x or x
  end
end
