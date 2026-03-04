-- Test: bitwise.lua - bit32 float conversion
-- From: bitwise.lua
-- What: Tests that bit32 operations properly convert floats to 32-bit unsigned integers

do
  -- Ensure bit32 is available
  package.preload.bit32 = package.preload.bit32 or function ()
    local bit = {}
    function bit.bnot (a) return ~a & 0xFFFFFFFF end
    function bit.bor (x, y, z, ...)
      if not z then return ((x or 0) | (y or 0)) & 0xFFFFFFFF
      else
        local arg = {...}
        local res = x | y | z
        for i = 1, #arg do res = res | arg[i] end
        return res & 0xFFFFFFFF
      end
    end
    return bit
  end

  local bit32 = require'bit32'
  assert(bit32.bor(3.0) == 3)
  assert(bit32.bor(-4.0) == 0xfffffffc)
  if 2.0^50 < 2.0^50 + 1.0 and 2.0^50 < (-1 >> 1) then
    assert(bit32.bor(2.0^32 - 5.0) == 0xfffffffb)
    assert(bit32.bor(-2.0^32 - 6.0) == 0xfffffffa)
  end
end
