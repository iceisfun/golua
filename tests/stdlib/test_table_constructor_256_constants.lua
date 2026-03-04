-- Test: constructs.lua - Table constructor with >256 constants
-- From: constructs.lua
-- What: Tests code generation when a table constructor uses more than 256 constants (bug since 5.4.0)

do
  local code = {"local x = {"}
  for i = 1, 257 do
    code[#code + 1] = i .. ".1,"
  end
  code[#code + 1] = "};"
  code = table.concat(code)
  local function check (ret, val)
    local code = code .. ret
    code = load(code)
    assert(code() == val)
  end
  check("return (1 ~ (2 or 3))", 1 ~ 2)
  check("return (1 | (2 or 3))", 1 | 2)
  check("return (1 + (2 or 3))", 1 + 2)
  check("return (1 << (2 or 3))", 1 << 2)
end
