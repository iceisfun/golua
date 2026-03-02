-- Test: coroutine.lua - Yields inside for iterators
-- From: coroutine.lua
-- What: Tests that coroutine.yield works from inside a for-in iterator function

do
  local function run(f, expectedYields)
    local co = coroutine.wrap(f)
    local results = {}
    local i = 1
    while true do
      local res = {co()}
      if #res == 0 then break end
      if res[1] == nil and res[2] then
        -- it's a yield
        assert(res[2] == expectedYields[i],
               "expected yield '" .. tostring(expectedYields[i]) ..
               "' but got '" .. tostring(res[2]) .. "'")
        i = i + 1
      else
        return res[1]
      end
    end
  end

  local f = function (s, i)
    if i%2 == 0 then coroutine.yield(nil, "for") end
    if i < s then return i + 1 end
  end

  assert(run(function ()
    local s = 0
    for i in f, 4, 0 do s = s + i end
    return s
  end, {"for", "for", "for"}) == 10)
end
