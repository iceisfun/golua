-- debug.traceback with negative level should produce an empty stack trace
-- (just the message and header), matching Lua 5.4.

local tb = debug.traceback("msg", -1)
print(tb == "msg\nstack traceback:")
--> =true
