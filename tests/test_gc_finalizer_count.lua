-- Bug: GC finalizer has off-by-one: the last created object's __gc
-- is not called. With 5 objects, only 4 get finalized.

local count = 0
for i = 1, 5 do
  setmetatable({id = i}, {
    __gc = function(self)
      count = count + 1
    end
  })
end

collectgarbage()
collectgarbage()

assert(count == 5, "all 5 objects should be finalized, got " .. count)

print("PASS")
