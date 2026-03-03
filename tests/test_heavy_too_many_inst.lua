-- Test: heavy.lua - Too many instructions
-- From: heavy.lua
-- What: Tests loading a chunk with an enormous number of assignment instructions

do
  local function loadrep (x, what)
    local p = 1<<20
    local s = string.rep(x, p)
    local count = 0
    local function f()
      count = count + p
      if count % (0x80*p) == 0 then
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

  print("loading chunk with too many instructions")
  local st, msg = loadrep("a = 10; ", "instructions")
  print('+')
end
