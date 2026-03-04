-- Test: errors.lua - Syntax limit tests
-- From: errors.lua
-- What: Tests that deeply nested expressions/statements fail with "too many" error

do
  local function testrep (init, rep, close, repc, finalresult)
    local s = init .. string.rep(rep, 100) .. close .. string.rep(repc, 100)
    local res, msg = load(s)
    assert(res)
    s = init .. string.rep(rep, 500)
    local res, msg = load(s)
    assert(not res and (string.find(msg, "too many") or
                        string.find(msg, "overflow")))
  end
  testrep("local a; a", ",a", "= 1", ",1")
  testrep("return ", "(", "2", ")", 2)
end
