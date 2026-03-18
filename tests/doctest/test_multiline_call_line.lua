-- Bug 4: Multi-line function call reports line of '(' not line of function name
local ok, err = pcall(load("local f=nil\nf\n()"))
print(err)
--> [string "local f=nil..."]:3: attempt to call a nil value (local 'f')
