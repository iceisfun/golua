-- Test: db.lua - Metamethod names in debug info
-- From: db.lua
-- What: Tests that debug.getinfo reports metamethod names correctly (namewhat == "metamethod")

do
  local a = {}
  local function f (t)
    local info = debug.getinfo(1);
    assert(info.namewhat == "metamethod")
    a.op = info.name
    return info.name
  end
  setmetatable(a, {
    __index = f; __add = f; __div = f; __mod = f; __concat = f; __pow = f;
    __mul = f; __idiv = f; __unm = f; __len = f; __sub = f;
    __eq = f; __le = f; __lt = f; __band = f; __bnot = f;
  })
  assert(a[3] == "index" and a^3 == "pow" and a..a == "concat")
end
