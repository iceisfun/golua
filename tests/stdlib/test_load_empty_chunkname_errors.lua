-- load with an empty chunk name formats compile diagnostics as [string ""].

local function msg(src) return select(2, load(src, "")) end
assert(msg("::l:: ::l::") == '[string ""]:1: label \'l\' already defined on line 1')
assert(msg("goto l") == '[string ""]:1: no visible label \'l\' for <goto> at line 1')
assert(msg("break") == '[string ""]:1: break outside loop near \'break\'')
assert(msg("local a <const> = 1; a = 2") == '[string ""]:1: attempt to assign to const variable \'a\'')
assert(msg("x = = 1") == '[string ""]:1: unexpected symbol near \'=\'')
assert(debug.getinfo(load("return 1", ""), "S").source == "")

print("OK")
