-- Test: heavy.lua - Huge identifier
-- From: heavy.lua
-- What: Tests that an extremely long identifier fails with "lexical element too long" or "not enough memory"

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

  local st, msg = loadrep("a", "chars")
  assert(not st and
    (string.find(msg, "lexical element too long") or
     string.find(msg, "not enough memory")))
end
