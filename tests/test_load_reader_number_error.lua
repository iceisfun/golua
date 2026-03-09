-- load(reader) stringification parity for numeric reader errors.

do
  local fn, err = load(function()
    error(42)
  end)

  assert(fn == nil)
  assert(type(err) == "string")
  assert(string.find(err, "42", 1, true) ~= nil,
    "numeric reader error should preserve numeric text")
  assert(string.find(err, "error object is a number value", 1, true) == nil)
end

do
  local ok, fn, err = pcall(function()
    return load(function()
      error(42)
    end)
  end)

  assert(ok == true)
  assert(fn == nil)
  assert(type(err) == "number")
  assert(err == 42)
end
