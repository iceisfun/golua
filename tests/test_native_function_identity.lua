do
  local s = tostring(print)
  assert(s:match("^function: 0x[0-9a-f]+$"), "native tostring should include pointer, got: " .. tostring(s))
  assert(string.format("%s", print) == s, "%%s should match tostring for native functions")
end

do
  local a = coroutine.wrap(function() end)
  local b = coroutine.wrap(function() end)
  local pa = string.format("%p", a)
  local pb = string.format("%p", b)
  assert(pa:match("^0x[0-9a-f]+$"), "native %%p should look like a pointer, got: " .. tostring(pa))
  assert(pb:match("^0x[0-9a-f]+$"), "native %%p should look like a pointer, got: " .. tostring(pb))
  assert(pa ~= pb, "distinct native closures should have distinct %%p identities")
end
