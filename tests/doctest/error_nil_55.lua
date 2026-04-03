-- Lua 5.5: error(nil) replaced by string message

-- error(nil) returns a string "<no error object>"
local ok, msg = pcall(error, nil)
print(ok, msg)
--> =false	<no error object>

-- error() with no args also returns the string
local ok2, msg2 = pcall(error)
print(ok2, msg2)
--> =false	<no error object>

-- the message is a string type
local ok3, msg3 = pcall(error, nil)
print(type(msg3), msg3)
--> =string	<no error object>

-- error(nil, 2) also produces the same message (no source location)
local ok4, msg4 = pcall(error, nil, 2)
print(type(msg4), msg4)
--> =string	<no error object>

-- error(nil, 0) also produces the same message
local ok5, msg5 = pcall(error, nil, 0)
print(type(msg5), msg5)
--> =string	<no error object>

-- non-nil errors still work normally
local ok6, msg6 = pcall(error, "hello")
print(ok6)
--> ~false

-- error(false) still passes false through (not nil)
local ok7, msg7 = pcall(error, false)
print(type(msg7), msg7)
--> =boolean	false

-- xpcall handler sees nil (replacement happens after handler)
local ok8, msg8 = xpcall(function() error(nil) end, function(e) return type(e) end)
print(ok8, msg8)
--> =false	nil

-- xpcall handler returning nil gets replaced
local ok9, msg9 = xpcall(function() error(nil) end, function(e) return nil end)
print(ok9, type(msg9), msg9)
--> =false	string	<no error object>

-- xpcall handler returning non-nil is kept
local ok10, msg10 = xpcall(function() error(nil) end, function(e) return "custom" end)
print(ok10, msg10)
--> =false	custom

-- assert(nil, nil) also gets the replacement (assert calls error internally)
local ok11, msg11 = pcall(assert, nil, nil)
print(type(msg11), msg11)
--> =string	<no error object>
