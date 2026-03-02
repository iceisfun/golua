-- Test: nextvar.lua - next function identity and global table traversal
-- From: nextvar.lua
-- What: Tests that next always uses the same iteration function, and that next/pairs can traverse the global table to find named globals.

do
  local nofind = {}

  a,b,c = 1,2,3
  a,b,c = nil

  -- next uses always the same iteraction function
  assert(next{} == next{})

  local function find (name)
    local n,v
    while 1 do
      n,v = next(_G, n)
      if not n then return nofind end
      assert(_G[n] ~= undef)
      if n == name then return v end
    end
  end

  local function find1 (name)
    for n,v in pairs(_G) do
      if n==name then return v end
    end
    return nil  -- not found
  end


  assert(print==find("print") and print == find1("print"))
  assert(_G["print"]==find("print"))
  assert(assert==find1("assert"))
  assert(nofind==find("return"))
  assert(not find1("return"))
  _G["ret" .. "urn"] = undef
  assert(nofind==find("return"))
  _G["xxx"] = 1
  assert(xxx==find("xxx"))
end
