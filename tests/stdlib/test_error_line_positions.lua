-- Test: errors.lua - Line error positions
-- From: errors.lua
-- What: Tests that runtime errors report the correct line number

do
  local function lineerror (s, l)
    local err,msg = pcall(load(s))
    local line = tonumber(string.match(msg, ":(%d+):"))
    assert(line == l or (not line and not l))
  end
  lineerror("local a\n for i=1,'a' do \n print(i) \n end", 2)
  lineerror("\n local a \n for k,v in 3 \n do \n print(k) \n end", 3)
end
