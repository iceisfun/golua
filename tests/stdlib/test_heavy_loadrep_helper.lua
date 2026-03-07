-- Test: heavy.lua - loadrep helper function
-- From: heavy.lua
-- What: Helper function used to feed repeated string patterns to load(), testing compiler limits.
--       Verifies that load() can accept a reader function and properly reports errors.

do
  local function loadrep (x, what)
    local p = 1<<20
    local s = string.rep(x, p)
    local count = 0
    local function f()
      count = count + p
      if count % (0x80*p) == 0 and io and io.stderr then
        io.stderr:write("(", count // 2^20, " M)")
      end
      return s
    end
    local st, msg = load(f, "=big")
    print("\nmemory: ", collectgarbage'count' * 1024)
    msg = string.match(msg, "^[^\n]+")
    print(string.format("total: 0x%x %s ('%s')", count, what, msg))
    return st, msg
  end

  -- Test that the helper works: feeding semicolons should be valid Lua
  -- but eventually hit some limit or succeed
  local st, msg = loadrep(";", "semicolons")
  -- The result depends on implementation limits, just verify it doesn't crash
  print("loadrep helper test completed")
end
