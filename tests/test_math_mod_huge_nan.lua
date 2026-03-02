-- Test: math.lua - Non-portable modulo with huge and NaN
-- From: math.lua
-- What: Tests modulo with math.huge and zero producing NaN, and modulo of finite values by math.huge.

do
  local function isNaN (x)
    return (x ~= x)
  end

  if not _port then
    local function anan (x) assert(isNaN(x)) end   -- assert Not a Number
    anan(0.0 % 0)
    anan(1.3 % 0)
    anan(math.huge % 1)
    anan(math.huge % 1e30)
    anan(-math.huge % 1e30)
    anan(-math.huge % -1e30)
    assert(1 % math.huge == 1)
    assert(1e30 % math.huge == 1e30)
    assert(1e30 % -math.huge == -math.huge)
    assert(-1 % math.huge == math.huge)
    assert(-1 % -math.huge == -1)
  end
end
