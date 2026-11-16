-- Test: invalid formats given to io.lines(filename, fmt...) are reported at
-- the argument position of the call (the filename is #1), like f:lines.
local name = os.tmpname()
local f = io.open(name, "w") f:write("1\n") f:close()

local function err(...)
  local args = table.pack(...)
  local ok, msg = pcall(function()
    for _ in io.lines(name, table.unpack(args, 1, args.n)) do end
  end)
  assert(not ok)
  return msg
end

assert(err('x'):find("bad argument #2 to 'for iterator' (invalid format)", 1, true))
assert(err('l', 'x'):find("bad argument #3 to 'for iterator' (invalid format)", 1, true))
assert(err({}):find("bad argument #2 to 'for iterator' (string expected, got table)", 1, true))

os.remove(name)
print("PASS")
