-- Test: events.lua - Protected metatable (__metatable)
-- From: events.lua
-- What: Tests __metatable metamethod to protect metatable from modification

do
  local a = {name = "NAME"}
  setmetatable(a, {__metatable = "xuxu",
                    __tostring=function(x) return x.name end})
  assert(getmetatable(a) == "xuxu")
  assert(tostring(a) == "NAME")
  assert(pcall(setmetatable, a, {}) == false)
  a.name = "gororoba"
  assert(tostring(a) == "gororoba")
end
