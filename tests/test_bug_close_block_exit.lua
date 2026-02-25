-- Bug #2: __close not called when do...end block exits normally

-- Plain do-end block: __close should fire on scope exit
local closed = false
do
  local x <close> = setmetatable({}, {__close = function() closed = true end})
end
assert(closed == true, "close not called on do-end block exit")

-- Multiple close vars: LIFO order (last declared closes first)
local order = {}
do
  local a <close> = setmetatable({n="a"}, {__close = function(s) order[#order+1] = s.n end})
  local b <close> = setmetatable({n="b"}, {__close = function(s) order[#order+1] = s.n end})
end
assert(order[1] == "b" and order[2] == "a", "close order wrong: expected b,a")

-- Nested blocks
local nested = {}
do
  local outer <close> = setmetatable({n="outer"}, {__close = function(s) nested[#nested+1] = s.n end})
  do
    local inner <close> = setmetatable({n="inner"}, {__close = function(s) nested[#nested+1] = s.n end})
  end
  -- inner should have closed here
  assert(nested[1] == "inner", "inner block close not called")
end
assert(nested[2] == "outer", "outer block close not called")
