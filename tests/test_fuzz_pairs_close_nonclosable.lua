-- test_fuzz_pairs_close_nonclosable:
-- Lua 5.5 validates the fourth value returned by __pairs as the to-be-closed
-- generic-for state. A non-closable value must fail before iteration.

local t = setmetatable({}, {
  __pairs = function()
    return function(_, control)
      if control == nil then
        return 1, "v"
      end
    end, "state", nil, "not closable"
  end,
})

local ok, err = pcall(function()
  for k, v in pairs(t) do
    error("iterator should not run: " .. tostring(k) .. " " .. tostring(v))
  end
end)

assert(ok == false, "pairs should reject a non-closable fourth __pairs result")
assert(type(err) == "string" and err:find("non%-closable value"),
  "expected non-closable value error, got: " .. tostring(err))

print("ok")
