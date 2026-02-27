-- Round 6 conformance: 6 bugs discovered via systematic probing
-- 1. table.remove rejects pos = #t + 1 (valid in Lua 5.4)
-- 2. __close error escapes pcall
-- 3. __close not called on goto block exit
-- 4. coroutine.close() doesn't trigger __close on TBC vars
-- 5. string.gsub rejects numeric replacement
-- 6. %b pattern fails when open == close characters

local pass, fail = 0, 0
local function check(name, ok, msg)
    if ok then
        pass = pass + 1
    else
        fail = fail + 1
        print("FAIL: " .. name .. ": " .. (msg or ""))
    end
end

-- 1. table.remove at #t + 1
local t1 = {1, 2, 3}
local ok1, r1 = pcall(table.remove, t1, 4)
check("table.remove #t+1", ok1, "should not error: " .. tostring(r1))
if ok1 then check("table.remove #t+1 nil", r1 == nil, "got: " .. tostring(r1)) end

local ok1b, r1b = pcall(table.remove, {}, 1)
check("table.remove empty pos 1", ok1b, "should not error: " .. tostring(r1b))

-- 2. __close error in pcall
local ok2, e2 = pcall(function()
    local x <close> = setmetatable({}, {__close = function() error("close_err") end})
end)
check("__close error caught by pcall", ok2 == false, "ok=" .. tostring(ok2))

-- 3. goto triggers __close
local goto_log = {}
do
    local x <close> = setmetatable({}, {__close = function()
        table.insert(goto_log, "closed")
    end})
    goto out
end
::out::
check("goto triggers __close", goto_log[1] == "closed",
    "got: " .. tostring(goto_log[1]))

-- 4. coroutine.close triggers __close
local co_log = {}
local co4 = coroutine.create(function()
    local x <close> = setmetatable({}, {__close = function()
        table.insert(co_log, "co_closed")
    end})
    coroutine.yield()
end)
coroutine.resume(co4)
coroutine.close(co4)
check("coroutine.close triggers __close", co_log[1] == "co_closed",
    "got: " .. tostring(co_log[1]))

-- 5. gsub with number replacement
local ok5, r5 = pcall(string.gsub, "x", "x", 42)
check("gsub number replacement", ok5, "should not error: " .. tostring(r5))
if ok5 then check("gsub number result", r5 == "42", "got: " .. tostring(r5)) end

-- 6. %b balanced with same char
local r6 = string.match("aXa", "%baa")
check("%b same char", r6 == "aXa", "got: " .. tostring(r6))

print(string.format("\nResults: %d passed, %d failed", pass, fail))
