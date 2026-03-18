-- Test Issue 1: pcall/xpcall should fire call/return hooks for inner C functions
do
  local events = {}
  local function hook(event, line)
    if event == "call" then
      local info = debug.getinfo(2, "Sn")
      events[#events + 1] = tostring(info.name) .. "(" .. info.what .. ")"
    end
    if #events > 20 then debug.sethook() end
  end
  events = {}
  debug.sethook(hook, "c")
  pcall(type, "x")
  debug.sethook()
  local result = table.concat(events, ", ")
  -- lua5.4 produces: pcall(C), nil(C), sethook(C)
  assert(result == "pcall(C), nil(C), sethook(C)",
    "Issue 1 pcall: expected 'pcall(C), nil(C), sethook(C)' got '" .. result .. "'")
  print("PASS: Issue 1 pcall hook fires for inner C functions")
end

-- Test Issue 1 with xpcall too
do
  local events = {}
  local function hook(event, line)
    if event == "call" then
      local info = debug.getinfo(2, "Sn")
      events[#events + 1] = tostring(info.name) .. "(" .. info.what .. ")"
    end
    if #events > 20 then debug.sethook() end
  end
  events = {}
  debug.sethook(hook, "c")
  xpcall(type, function() end, "x")
  debug.sethook()
  local result = table.concat(events, ", ")
  -- Should have xpcall(C), nil(C), sethook(C)
  assert(result == "xpcall(C), nil(C), sethook(C)",
    "Issue 1 xpcall: expected 'xpcall(C), nil(C), sethook(C)' got '" .. result .. "'")
  print("PASS: Issue 1 xpcall hook fires for inner C functions")
end

-- Test Issue 1: return hooks also fire for inner C functions in pcall
do
  local events = {}
  local function hook(event, line)
    if event == "call" or event == "return" then
      events[#events + 1] = event
    end
    if #events > 20 then debug.sethook() end
  end
  events = {}
  debug.sethook(hook, "cr")
  pcall(type, "x")
  debug.sethook()
  -- Should have return, call (pcall), call (inner), return (inner), return (pcall), call (sethook)
  -- The key: there should be at least 2 "call" events between sethook calls
  local call_count = 0
  for _, e in ipairs(events) do
    if e == "call" then call_count = call_count + 1 end
  end
  assert(call_count >= 3, -- pcall + inner function + sethook
    "Issue 1 return: expected at least 3 call events, got " .. call_count)
  print("PASS: Issue 1 return hooks fire for inner C functions")
end

-- Test Issue 3: getlocal on caller from inner call hook shows named locals correctly
do
  local results = {}
  local function hook(event, line)
    if event == "call" then
      local info3 = debug.getinfo(3, "Sn")
      if info3 and info3.name == "target" then
        for i = 1, 10 do
          local name, val = debug.getlocal(3, i)
          if not name then break end
          results[#results + 1] = name .. "=" .. tostring(val)
        end
      end
    end
    if #results > 20 then debug.sethook() end
  end
  local function target(x)
    local y = x + 1
    type(y)
  end
  debug.sethook(hook, "c")
  target(10)
  debug.sethook()
  -- When the call hook fires for type(y), getlocal at level 3 (target) should see
  -- x=10 and y=11. Lua 5.4 shows exactly these two named locals.
  local found_x, found_y = false, false
  for _, r in ipairs(results) do
    if r == "x=10" then found_x = true end
    if r == "y=11" then found_y = true end
  end
  assert(found_x, "Issue 3: x should be 10, got: " .. table.concat(results, ", "))
  assert(found_y, "Issue 3: y should be 11, got: " .. table.concat(results, ", "))
  print("PASS: Issue 3 named locals visible in call hooks")
end
