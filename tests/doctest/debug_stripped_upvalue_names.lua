-- Stripped upvalue names should be "(no name)" not ""
local prog = [[local a = 12; local f = function() return a end; return f]]
local f = assert(load(string.dump(assert(load(prog)), true)))()
local name = debug.getupvalue(f, 1)
print(name)                            --> (no name)

-- Non-stripped should still show real name
local f2 = assert(load(string.dump(assert(load(prog)))))()
local name2 = debug.getupvalue(f2, 1)
print(name2)                           --> a

-- setupvalue on stripped function returns "(no name)"
print(debug.setupvalue(f, 1, 99))      --> (no name)
