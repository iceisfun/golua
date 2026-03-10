do
  local old = debug.getmetatable(0)
  debug.setmetatable(0, {
    __pairs = function()
      return next, {a = 1}, nil
    end,
  })

  local out = {}
  for k, v in pairs(0) do
    out[k] = v
  end
  assert(out.a == 1)
  debug.setmetatable(0, old)
end

do
  local co = coroutine.create(function() end)
  local ok, err = pcall(next, co)
  assert(not ok)
  assert(type(err) == "string" and string.find(err, "table expected, got thread", 1, true), tostring(err))
end

do
  local co = coroutine.create(function() end)
  local ok, err = pcall(function()
    for _k, _v in pairs(co) do
    end
  end)
  assert(not ok)
  assert(type(err) == "string" and string.find(err, "table expected, got thread", 1, true), tostring(err))
end

do
  local co = coroutine.create(function() end)
  local old = debug.getmetatable(co)
  debug.setmetatable(co, {
    __pairs = function()
      return next, {x = 3}, nil
    end,
  })

  local seen = {}
  for k, v in pairs(co) do
    seen[k] = v
  end
  assert(seen.x == 3)
  debug.setmetatable(co, old)
end
