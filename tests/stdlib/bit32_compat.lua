-- Test: bitwise.lua - bit32 compatibility library
-- From: bitwise.lua
-- What: Implements and tests a bit32 library using native bitwise operators (band, bor, bxor, bnot, lshift, rshift, arshift, lrotate, rrotate, extract, replace)

do
  package.preload.bit32 = function ()
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
    function bit.lshift (a, b)
      if b >= 0 then return (a << b) & 0xFFFFFFFF
      else return (a & 0xFFFFFFFF) >> -b end
    end
    function bit.rshift (a, b)
      if b >= 0 then return (a & 0xFFFFFFFF) >> b
      else return (a << -b) & 0xFFFFFFFF end
    end
    function bit.arshift (a, b)
      a = a & 0xFFFFFFFF
      if b <= 0 then return (a << -b) & 0xFFFFFFFF end
      local m = (a & 0x80000000) ~= 0
      a = a >> b
      if m then a = a | ~(0xFFFFFFFF >> b) end
      return a & 0xFFFFFFFF
    end
    function bit.lrotate (a, b)
      a = a & 0xFFFFFFFF
      b = b & 31
      return ((a << b) | (a >> (32 - b))) & 0xFFFFFFFF
    end
    function bit.rrotate (a, b)
      return bit.lrotate(a, -b)
    end
    function bit.extract (a, f, w)
      w = w or 1
      assert(f >= 0 and w > 0 and f + w <= 32,
             "trying to access non-existent bits")
      return (a >> f) & ~(-1 << w)
    end
    function bit.replace (a, v, f, w)
      w = w or 1
      assert(f >= 0 and w > 0 and f + w <= 32,
             "trying to access non-existent bits")
      local mask = ~(-1 << w)
      return (a & ~(mask << f)) | ((v & mask) << f)
    end
    return bit
  end

  local bit32 = require'bit32'
  assert(bit32.band() == bit32.bnot(0))
  assert(bit32.btest() == true)
  assert(bit32.bor() == 0)
  assert(bit32.bxor() == 0)
end
