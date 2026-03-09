-- package.searchpath should not skip empty templates between separators.

do
  local ok, found, err = pcall(package.searchpath, "a.b", ";")
  assert(ok)
  assert(found == nil)
  assert(type(err) == "string")
  -- Lua 5.4 emits one "no file ''" entry per empty template.
  -- For path ";" there are two empty templates.
  local _, count = string.gsub(err, "no file ''", "")
  assert(count == 2, "expected 2 empty-template entries, got " .. tostring(count) .. ": " .. err)

  ok, found, err = pcall(package.searchpath, "x", "?.lua;;?/init.lua")
  assert(ok)
  assert(found == nil)
  assert(type(err) == "string")
  local p1 = string.find(err, "no file 'x.lua'", 1, true)
  local p2 = string.find(err, "\n\tno file ''", 1, true)
  local p3 = string.find(err, "\n\tno file 'x/init.lua'", 1, true)
  assert(p1 ~= nil and p2 ~= nil and p3 ~= nil and p1 < p2 and p2 < p3, err)
end
