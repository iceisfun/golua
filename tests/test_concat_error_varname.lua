-- BUG: Concatenation error messages don't include variable name context
-- Reference Lua 5.4: "attempt to concatenate a table value (local 'myvar')"
-- GoLua:             "attempt to concatenate a table value"
-- Note: arithmetic errors DO include variable names correctly; only concat is affected.

-- Local variable
local function test_local()
  local myvar = {}
  return myvar .. "x"
end
local ok, err = pcall(test_local)
assert(not ok)
assert(err:find("%(local 'myvar'%)"),
  "should include (local 'myvar'), got: " .. err)

-- Upvalue
local myup = {}
local function test_upvalue()
  return myup .. "x"
end
local ok2, err2 = pcall(test_upvalue)
assert(not ok2)
assert(err2:find("%(upvalue 'myup'%)"),
  "should include (upvalue 'myup'), got: " .. err2)

-- Global
someGlobal = {}
local function test_global()
  return someGlobal .. "x"
end
local ok3, err3 = pcall(test_global)
assert(not ok3)
assert(err3:find("%(global 'someGlobal'%)"),
  "should include (global 'someGlobal'), got: " .. err3)

