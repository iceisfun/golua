-- Test: table.move with metamethods
-- From: sort.lua
-- What: Tests table.move interacting with __index and __newindex metamethods, including error propagation from metamethods.

do
  local function eqT (a, b)
    for k, v in pairs(a) do assert(b[k] == v) end
    for k, v in pairs(b) do assert(a[k] == v) end
  end

  local a = setmetatable({}, {
        __index = function (_,k) return k * 10 end,
        __newindex = error})
  local b = table.move(a, 1, 10, 3, {})
  eqT(a, {})
  eqT(b, {nil,nil,10,20,30,40,50,60,70,80,90,100})

  b = setmetatable({""}, {
        __index = error,
        __newindex = function (t,k,v)
          t[1] = string.format("%s(%d,%d)", t[1], k, v)
      end})
  table.move(a, 10, 13, 3, b)
  assert(b[1] == "(3,100)(4,110)(5,120)(6,130)")
  local stat, msg = pcall(table.move, b, 10, 13, 3, b)
  assert(not stat and msg == b)
end
