-- Test: errors.lua - Goto/break error messages
-- From: errors.lua
-- What: Tests error messages for duplicate labels and invisible labels

do
  local function checksyntax (prog, msg, line)
    local st, err = load(prog)
    assert(string.find(err, "line " .. line))
    assert(string.find(err, msg, 1, true))
  end
  checksyntax([[
    ::A:: a = 1
    ::A::
  ]], "label 'A' already defined", 1)
  checksyntax([[
    a = 1
    goto A
    do ::A:: end
  ]], "no visible label 'A'", 2)
end
