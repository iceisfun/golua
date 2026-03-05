-- Test: heavy.lua - Too many lines
-- From: heavy.lua
-- What: Tests that loading a chunk with an extremely large number of newlines fails with "too many lines"

do
  local function loadrep (x, what)
    local p = 1<<20
    local s = string.rep(x, p)
    local count = 0
    local function f()
      count = count + p
      return s
    end
    local st, msg = load(f, "=big")
    print("\nmemory: ", collectgarbage'count' * 1024)
    msg = string.match(msg, "^[^\n]+")
    print(string.format("total: 0x%x %s ('%s')", count, what, msg))
    return st, msg
  end

  print("loading chunk with too many lines")
  local st, msg = loadrep("\n", "lines")
  assert(not st and string.find(msg, "too many lines"))
  print('+')
end
