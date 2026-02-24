-- Test: coroutine chain (centipede pattern)
-- Each coroutine resumes the next, passing a growing string.

-- Small chain first (3 deep)
local function make_link(next_co, id)
    return coroutine.create(function(val)
        if next_co then
            local ok, result = coroutine.resume(next_co, val .. ">" .. id)
            assert(ok, "inner resume failed: " .. tostring(result))
            coroutine.yield(result)
        else
            coroutine.yield(val .. ">" .. id)
        end
    end)
end

-- 3-link chain
local c3 = nil
for i = 3, 1, -1 do
    c3 = make_link(c3, tostring(i))
end
local ok3, val3 = coroutine.resume(c3, "start")
assert(ok3, "3-chain resume failed: " .. tostring(val3))
assert(val3 == "start>1>2>3",
    "3-chain expected 'start>1>2>3', got: " .. tostring(val3))

-- 10-link chain
local c10 = nil
for i = 10, 1, -1 do
    c10 = make_link(c10, tostring(i))
end
local ok10, val10 = coroutine.resume(c10, "go")
assert(ok10, "10-chain resume failed: " .. tostring(val10))
assert(val10 == "go>1>2>3>4>5>6>7>8>9>10",
    "10-chain expected 'go>1>2>3>4>5>6>7>8>9>10', got: " .. tostring(val10))

-- 50-link chain (the full centipede)
local c50 = nil
for i = 50, 1, -1 do
    c50 = make_link(c50, tostring(i))
end
local ok50, val50 = coroutine.resume(c50, "start")
assert(ok50, "50-chain resume failed: " .. tostring(val50))
assert(type(val50) == "string", "result should be string")
-- Check start and end of the string
assert(val50:sub(1, 7) == "start>1",
    "50-chain should start with 'start>1', got: " .. val50:sub(1, 20))
assert(val50:sub(-3) == ">50",
    "50-chain should end with '>50', got: " .. val50:sub(-10))
