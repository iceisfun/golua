-- Test: bitwise.lua - bit32 special cases and rotations
-- From: bitwise.lua
-- What: Tests bit32 operations with special values (0, max values, patterns) and rotation edge cases

do
  -- Ensure bit32 is available
  package.preload.bit32 = package.preload.bit32 or function ()
    local bit = {}
    function bit.bnot (a) return ~a & 0xFFFFFFFF end
    function bit.band (x, y, z, ...)
      if not z then return ((x or -1) & (y or -1)) & 0xFFFFFFFF
      else
        local arg = {...}
        local res = x & y & z
        for i = 1, #arg do res = res & arg[i] end
        return res & 0xFFFFFFFF
      end
    end
    function bit.bor (x, y, z, ...)
      if not z then return ((x or 0) | (y or 0)) & 0xFFFFFFFF
      else
        local arg = {...}
        local res = x | y | z
        for i = 1, #arg do res = res | arg[i] end
        return res & 0xFFFFFFFF
      end
    end
    function bit.bxor (x, y, z, ...)
      if not z then return ((x or 0) ~ (y or 0)) & 0xFFFFFFFF
      else
        local arg = {...}
        local res = x ~ y ~ z
        for i = 1, #arg do res = res ~ arg[i] end
        return res & 0xFFFFFFFF
      end
    end
    function bit.btest (...)
      return bit.band(...) ~= 0
    end
    function bit.lrotate (a, b)
      a = a & 0xFFFFFFFF
      b = b & 31
      return ((a << b) | (a >> (32 - b))) & 0xFFFFFFFF
    end
    function bit.rrotate (a, b)
      return bit.lrotate(a, -b)
    end
    return bit
  end

  local bit32 = require'bit32'
  local c = {0, 1, 2, 3, 10, 0x80000000, 0xaaaaaaaa, 0x55555555,
             0xffffffff, 0x7fffffff}
  for _, b in pairs(c) do
    assert(bit32.band(b) == b)
    assert(bit32.band(b, b) == b)
    assert(bit32.btest(b, b) == (b ~= 0))
    assert(bit32.band(b, bit32.bnot(b)) == 0)
    assert(bit32.bor(b, bit32.bnot(b)) == bit32.bnot(0))
    assert(bit32.bxor(b, b) == 0)
    assert(bit32.bnot(bit32.bnot(b)) == b)
    assert(bit32.lrotate(b, 32) == b)
    assert(bit32.rrotate(b, 32) == b)
  end
end
