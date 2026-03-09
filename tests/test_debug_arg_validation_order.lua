-- Argument validation order parity for debug library helpers.

local function expect_arg(err, n, fname)
  local msg = tostring(err)
  assert(string.find(msg, "bad argument #" .. tostring(n), 1, true) ~= nil,
    "wrong arg index: " .. msg)
  assert(string.find(msg, "'" .. fname .. "'", 1, true) ~= nil,
    "wrong function name: " .. msg)
end

do
  local ok, err = pcall(debug.getupvalue, nil, "x")
  assert(ok == false)
  expect_arg(err, 2, "debug.getupvalue")
end

do
  local ok, err = pcall(debug.setupvalue, nil, "x", 1)
  assert(ok == false)
  expect_arg(err, 2, "debug.setupvalue")
end

do
  local ok, err = pcall(debug.setupvalue)
  assert(ok == false)
  expect_arg(err, 3, "debug.setupvalue")
  assert(string.find(tostring(err), "value expected", 1, true) ~= nil)
end

do
  local ok, err = pcall(debug.setlocal, 0, 1)
  assert(ok == false)
  expect_arg(err, 3, "debug.setlocal")
  assert(string.find(tostring(err), "value expected", 1, true) ~= nil)
end

do
  local ok, err = pcall(debug.upvalueid, nil, "x")
  assert(ok == false)
  expect_arg(err, 2, "debug.upvalueid")
end

do
  local ok, err = pcall(debug.upvaluejoin, "x", {}, "x", 1)
  assert(ok == false)
  expect_arg(err, 2, "debug.upvaluejoin")
end

do
  local x = 1
  local function f1() return x end
  local ok, err = pcall(debug.upvaluejoin, f1, 1, nil, {})
  assert(ok == false)
  expect_arg(err, 4, "debug.upvaluejoin")
end

do
  local ok, err = pcall(debug.getlocal, {}, function() end, 0)
  assert(ok == false)
  expect_arg(err, 2, "debug.getlocal")
end

do
  local ok, err = pcall(debug.setlocal, 1, false, {}, nil)
  assert(ok == false)
  expect_arg(err, 2, "debug.setlocal")
  assert(string.find(tostring(err), "got boolean", 1, true) ~= nil)
end

do
  local ok, err = pcall(debug.getlocal, 1.5, 1, false)
  assert(ok == false)
  expect_arg(err, 1, "debug.getlocal")
  assert(string.find(tostring(err), "number has no integer representation", 1, true) ~= nil)
end

do
  local ok, err = pcall(debug.sethook, true)
  assert(ok == false)
  expect_arg(err, 2, "debug.sethook")
  assert(string.find(tostring(err), "got no value", 1, true) ~= nil)
end

do
  local ok, err = pcall(debug.sethook, true, 1)
  assert(ok == false)
  expect_arg(err, 1, "debug.sethook")
end

do
  local ok, err = pcall(debug.sethook, function() end, "", 1.5)
  assert(ok == false)
  expect_arg(err, 3, "debug.sethook")
  assert(string.find(tostring(err), "number has no integer representation", 1, true) ~= nil)
end

do
  debug.sethook(function() end, "l")
  local h = debug.gethook()
  assert(type(h) == "function")

  -- Unknown mask chars produce an empty mask, which clears the hook.
  debug.sethook(function() end, "x")
  local h2, m2, c2 = debug.gethook()
  assert(h2 == nil and m2 == nil and c2 == nil)
end
