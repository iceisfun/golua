-- Test: bitwise.lua - bit32 extract/replace
-- From: bitwise.lua
-- What: Tests bit field extraction and replacement operations

do
  -- Ensure bit32 is available
  package.preload.bit32 = package.preload.bit32 or function ()
    local bit = {}
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
  assert(bit32.extract(0x12345678, 0, 4) == 8)
  assert(bit32.extract(0x12345678, 4, 4) == 7)
  assert(bit32.extract(0xa0001111, 28, 4) == 0xa)
  assert(bit32.extract(0xf2345679, 0, 32) == 0xf2345679)
  assert(not pcall(bit32.extract, 0, -1))
  assert(not pcall(bit32.extract, 0, 32))

  assert(bit32.replace(0x12345678, 5, 28, 4) == 0x52345678)
  assert(bit32.replace(0x12345678, 0x87654321, 0, 32) == 0x87654321)
end
