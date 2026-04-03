-- Test debug.getinfo extraargs field (Lua 5.5)
-- extraargs is reported under the 't' option and counts
-- extra arguments prepended by __call metamethod chains.

-- Normal function call has 0 extraargs
local function f()
  local info = debug.getinfo(1, "t")
  return info.extraargs
end
print(f()) --> 0

-- __call metamethod: handler receives 1 extra arg (self)
local function handler(self, ...)
  local info = debug.getinfo(1, "t")
  return info.extraargs
end

local a = setmetatable({}, {__call = handler})
print(a(10, 20)) --> 1

-- Chained __call: b's __call is a, so handler gets 2 extra args
local b = setmetatable({}, {__call = a})
print(b(10, 20)) --> 2

-- Triple chain: c -> b -> a -> handler, 3 extra args
local c = setmetatable({}, {__call = b})
print(c(10, 20)) --> 3

-- Tailcall inherits extraargs from caller's frame
local function f2(...)
  local info = debug.getinfo(1, "t")
  return info.extraargs, info.istailcall
end

local t2 = setmetatable({}, {
  __call = function(self, ...)
    return f2(...) -- tailcall
  end
})
local ea, tc = t2(1, 2, 3)
print(ea, tc) --> 1	true

-- Non-tailcall does NOT inherit extraargs
local t3 = setmetatable({}, {
  __call = function(self, ...)
    local r = f2(...)  -- not a tailcall
    return r
  end
})
print(t3(1, 2, 3)) --> 0
