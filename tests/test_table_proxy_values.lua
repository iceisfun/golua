do
  local old_num_mt = debug.getmetatable(0)
  local data = { [1] = "b", [2] = "a" }

  debug.setmetatable(0, {
    __len = function()
      return #data
    end,
    __index = function(_, k)
      return data[k]
    end,
    __newindex = function(_, k, v)
      data[k] = v
    end,
  })

  assert(table.concat(0, ",") == "b,a")
  table.insert(0, "c")
  assert(data[1] == "b" and data[2] == "a" and data[3] == "c")
  assert(table.remove(0, 1) == "b")
  table.sort(0)
  assert(data[1] == "a" and data[2] == "c")

  debug.setmetatable(0, old_num_mt)
end

do
  local old_num_mt = debug.getmetatable(0)
  local dst = coroutine.create(function() end)
  local old_thread_mt = debug.getmetatable(dst)
  local src = { [1] = 10, [2] = 20 }
  local out = {}

  debug.setmetatable(0, {
    __index = function(_, k)
      return src[k]
    end,
  })
  debug.setmetatable(dst, {
    __newindex = function(_, k, v)
      out[k] = v
    end,
  })

  assert(table.move(0, 1, 2, 3, dst) == dst)
  assert(out[3] == 10 and out[4] == 20)

  debug.setmetatable(0, old_num_mt)
  debug.setmetatable(dst, old_thread_mt)
end

do
  local old_num_mt = debug.getmetatable(0)
  debug.setmetatable(0, {
    __len = function()
      return 1
    end,
  })

  local ok, err = pcall(function()
    return table.concat(0, ",")
  end)
  assert(not ok)
  assert(type(err) == "string" and string.find(err, "table expected, got number", 1, true), tostring(err))

  debug.setmetatable(0, old_num_mt)
end
