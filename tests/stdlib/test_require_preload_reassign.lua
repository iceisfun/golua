-- require should keep using the originally captured package.preload table,
-- so reassigning package.preload does not affect preload lookup.

do
  package.loaded = {}
  package.preload = {}
  package.preload.m = function()
    error("P")
  end

  local ok, err = pcall(require, "m")
  assert(ok == false)
  local msg = tostring(err)
  assert(string.find(msg, "module 'm' not found", 1, true) ~= nil,
    "require should ignore reassigned preload table: " .. msg)
end
