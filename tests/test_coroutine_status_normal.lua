-- Bug: When coroutine A resumes coroutine B, calling
-- coroutine.status(A) from within B should return "normal",
-- but golua returns "running".

local co1, co2
local status_of_co2 = nil

co1 = coroutine.create(function()
  -- Called from co2. co2 should be "normal" (it resumed us).
  status_of_co2 = coroutine.status(co2)
end)

co2 = coroutine.create(function()
  coroutine.resume(co1)
end)

coroutine.resume(co2)

assert(status_of_co2 == "normal",
  "coroutine that resumed another should have status 'normal', got '" .. tostring(status_of_co2) .. "'")

print("PASS")
