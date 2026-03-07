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
    msg = string.match(msg, "^[^\n]+")
    return st, msg
  end

  local st, msg = loadrep("\n", "lines")
  assert(not st and string.find(msg, "too many lines"))
end
