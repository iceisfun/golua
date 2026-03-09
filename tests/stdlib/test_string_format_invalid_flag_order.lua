-- string.format should validate argument availability/type before rejecting
-- invalid flag combinations for u/o/x/X conversions.

do
  local ok, err = pcall(string.format, "%+1.0u")
  assert(ok == false)
  local msg = tostring(err)
  assert(string.find(msg, "bad argument #2", 1, true) ~= nil, msg)
  assert(string.find(msg, "no value", 1, true) ~= nil, msg)
end

do
  local ok, err = pcall(string.format, "%+1.0u", "x")
  assert(ok == false)
  local msg = tostring(err)
  assert(string.find(msg, "bad argument #2", 1, true) ~= nil, msg)
  assert(string.find(msg, "number expected", 1, true) ~= nil, msg)
end

do
  local ok, err = pcall(string.format, "% .3o", 1/0)
  assert(ok == false)
  local msg = tostring(err)
  assert(string.find(msg, "bad argument #2", 1, true) ~= nil, msg)
  assert(string.find(msg, "number has no integer representation", 1, true) ~= nil, msg)
end

do
  local ok, err = pcall(string.format, "%+2x", 10)
  assert(ok == false)
  local msg = tostring(err)
  assert(string.find(msg, "invalid conversion specification", 1, true) ~= nil, msg)
end

do
  local ok, err = pcall(string.format, "%#1.1d", false)
  assert(ok == false)
  local msg = tostring(err)
  assert(string.find(msg, "bad argument #2", 1, true) ~= nil, msg)
  assert(string.find(msg, "number expected", 1, true) ~= nil, msg)
end

do
  local ok, err = pcall(string.format, "%#1.1d", 7)
  assert(ok == false)
  local msg = tostring(err)
  assert(string.find(msg, "invalid conversion specification", 1, true) ~= nil, msg)
end

do
  local ok, err = pcall(string.format, "%+#.0c", false)
  assert(ok == false)
  local msg = tostring(err)
  assert(string.find(msg, "invalid conversion specification", 1, true) ~= nil, msg)
end

do
  local ok, err = pcall(string.format, "%..3q", 1)
  assert(ok == false)
  local msg = tostring(err)
  assert(string.find(msg, "specifier '%q' cannot have modifiers", 1, true) ~= nil, msg)
end

do
  assert(string.format("%+.0i", 0) == "+")
  assert(string.format("% .0d", 0) == " ")
end

do
  -- Dot with no precision digits means precision 0 for %a/%A.
  assert(string.format("%.A", 255) == "0X2P+7")
  assert(string.format("%.a", -2.25) == "-0x1p+1")
end

do
  -- Zero-padding for %a/%A should pad after sign/prefix, including space-sign.
  assert(string.format("% 010.0a", 1) == " 0x0001p+0")
end

do
  -- Lua's %p for empty string is not "(null)".
  assert(string.format("%p", "") ~= "(null)")
end
