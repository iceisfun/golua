-- Multiple indexed assignment where the RHS is a multi-return call and the
-- targets dispatch through __newindex. The __newindex metamethod is itself a
-- function call, which must NOT clobber the still-pending value registers that
-- hold the (right-to-left) un-stored results. Keys are function calls so the
-- value registers land above the key temps (the layout that exposed the bug).
local log = {}
local function L(s) log[#log+1] = s end

local mt = setmetatable({}, {
  __newindex = function(_, k, v) L("store " .. tostring(k) .. "=" .. tostring(v)) end,
})
local function K(x) return x end
local function MV() return 100, 200, 300 end

mt[K(1)], mt[K(2)], mt[K(3)] = MV()
print(table.concat(log, ","))
--> =store 3=300,store 2=200,store 1=100

-- Fewer returns than targets: extra targets get nil, middle value preserved.
local log2 = {}
local function L2(s) log2[#log2+1] = s end
local mt2 = setmetatable({}, {
  __newindex = function(_, k, v) L2("store " .. tostring(k) .. "=" .. tostring(v)) end,
})
local function MV2() return 11, 22 end
mt2[K(1)], mt2[K(2)], mt2[K(3)] = MV2()
print(table.concat(log2, ","))
--> =store 3=nil,store 2=22,store 1=11
