-- Test __index chain depth matches Lua 5.4 (MAXTAGLOOP=2000)
local function make_chain(depth)
  local t = {}
  for i = 1, depth do
    t = setmetatable({}, {__index = t})
  end
  return t
end

-- Depth 2000 should succeed
local chain = make_chain(2000)
chain_base = chain
-- Walk to find the base
local c = chain
for i = 1, 2000 do
  c = getmetatable(c).__index
end
-- Set value on base
c.x = 42
-- Access through 2000-deep chain should work
local ok, val = pcall(function() return chain.x end)
assert(ok, "depth 2000 should work, got: " .. tostring(val))
assert(val == 42, "expected 42, got " .. tostring(val))

-- Depth 2001 should error
local chain2 = make_chain(2001)
local c2 = chain2
for i = 1, 2001 do
  c2 = getmetatable(c2).__index
end
c2.x = 99
local ok2, err2 = pcall(function() return chain2.x end)
assert(not ok2, "depth 2001 should error")
assert(tostring(err2):find("too long"), "expected 'too long' error, got: " .. tostring(err2))

print("PASS")
