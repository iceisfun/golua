-- Test: Line ending normalization
-- From: literals.lua
-- What: Tests that different line endings (\n, \r, \n\r, \r\n) are all normalized correctly in loaded chunks, and that line counting works across all line-end variants.

do
  local function dostring (x) return assert(load(x), "")() end

  local prog = [[
local a = 1        -- a comment
local b = 2


x = [=[
hi
]=]
y = "\
hello\r\n\
"
return require"debug".getinfo(1).currentline
]]

  for _, n in pairs{"\n", "\r", "\n\r", "\r\n"} do
    local prog, nn = string.gsub(prog, "\n", n)
    assert(dostring(prog) == nn)
    assert(_G.x == "hi\n" and _G.y == "\nhello\r\n\n")
  end
  _G.x, _G.y = nil
end
