-- os.setlocale argument/category validation parity checks.

do
  local ok, err = pcall(os.setlocale, true)
  assert(ok == false)
  local msg = tostring(err)
  assert(string.find(msg, "bad argument #1", 1, true) ~= nil, msg)
  assert(string.find(msg, "string expected", 1, true) ~= nil, msg)
end

do
  local ok, res = pcall(os.setlocale, 1)
  assert(ok == true)
  assert(res == nil)
end

do
  local ok, err = pcall(os.setlocale, "C", true)
  assert(ok == false)
  local msg = tostring(err)
  assert(string.find(msg, "bad argument #2", 1, true) ~= nil, msg)
  assert(string.find(msg, "string expected", 1, true) ~= nil, msg)
end

do
  local ok, err = pcall(os.setlocale, "C", 1)
  assert(ok == false)
  local msg = tostring(err)
  assert(string.find(msg, "bad argument #2", 1, true) ~= nil, msg)
  assert(string.find(msg, "invalid option '1'", 1, true) ~= nil, msg)
end
