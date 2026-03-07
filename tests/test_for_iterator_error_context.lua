-- BUG: for-in iterator call error messages missing "(for iterator 'for iterator')" context
-- Reference Lua 5.4: "attempt to call a number value (for iterator 'for iterator')"
-- GoLua:             "attempt to call a number value"

-- Non-callable iterator (number)
local function test1()
  for k in 42 do end
end
local ok, err = pcall(test1)
assert(not ok)
assert(err:find("for iterator"),
  "should include 'for iterator' context, got: " .. err)

-- Non-callable iterator (nil)
local function test2()
  for k in nil do end
end
local ok2, err2 = pcall(test2)
assert(not ok2)
assert(err2:find("for iterator"),
  "should include 'for iterator' context, got: " .. err2)

-- Non-callable iterator (boolean)
local function test3()
  for k in true do end
end
local ok3, err3 = pcall(test3)
assert(not ok3)
assert(err3:find("for iterator"),
  "should include 'for iterator' context, got: " .. err3)

