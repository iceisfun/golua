-- Argument validation order and error naming parity for math APIs.

do
  local ok, err = pcall(math.fmod)
  assert(ok == false)
  local msg = tostring(err)
  assert(string.find(msg, "bad argument #2", 1, true) ~= nil, msg)
  assert(string.find(msg, "math.fmod", 1, true) ~= nil, msg)
end

do
  local ok, err = pcall(math.fmod, "x")
  assert(ok == false)
  local msg = tostring(err)
  assert(string.find(msg, "bad argument #2", 1, true) ~= nil, msg)
end

do
  local ok, err = pcall(math.atan)
  assert(ok == false)
  local msg = tostring(err)
  assert(string.find(msg, "math.atan", 1, true) ~= nil, msg)
end

do
  local ok, err = pcall(math.atan, false)
  assert(ok == false)
  local msg = tostring(err)
  assert(string.find(msg, "math.atan", 1, true) ~= nil, msg)
end

do
  local ok, err = pcall(math.atan, 1, "x")
  assert(ok == false)
  local msg = tostring(err)
  assert(string.find(msg, "bad argument #2", 1, true) ~= nil, msg)
  assert(string.find(msg, "math.atan", 1, true) ~= nil, msg)
end

do
  assert(tostring(math.asin(-1/0)) == "nan")
end

do
  assert(tostring(math.asin(0/0)) == "-nan")
  assert(tostring(math.acos(0/0)) == "-nan")
end
