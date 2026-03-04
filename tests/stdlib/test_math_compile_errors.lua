-- Test: math.lua - Compile-time error avoidance
-- From: math.lua
-- What: Tests that certain operations (divide by zero, float-to-int conversion errors, bitwise on huge) produce runtime errors when compiled as code strings.

do
  local maxint <const> = math.maxinteger
  local intbits <const> = math.floor(math.log(maxint, 2) + 0.5) + 1

  local function checkerror (msg, f, ...)
    local s, err = pcall(f, ...)
    assert(not s and string.find(err, msg))
  end

  local msgf2i = "number.* has no integer representation"

  local function checkcompt (msg, code)
    checkerror(msg, assert(load(code)))
  end
  checkcompt("divide by zero", "return 2 // 0")
  checkcompt(msgf2i, "return 2.3 >> 0")
  checkcompt(msgf2i, ("return 2.0^%d & 1"):format(intbits - 1))
  checkcompt("field 'huge'", "return math.huge << 1")
  checkcompt(msgf2i, ("return 1 | 2.0^%d"):format(intbits - 1))
  checkcompt(msgf2i, "return 2.3 ~ 0.0")
end
