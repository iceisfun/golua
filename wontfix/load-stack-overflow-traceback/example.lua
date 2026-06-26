-- load() of a too-deeply-nested chunk: the parser hits its recursion limit and
-- returns nil + "C stack overflow". Reference Lua, when load() is called
-- DIRECTLY (not under pcall), embeds a stack traceback into the error MESSAGE
-- string it returns; golua returns the clean message.

local deep = "return " .. ("("):rep(5000) .. "1" .. (")"):rep(5000)
print(load(deep, "=t"))
--> golua:     nil   C stack overflow
--> lua5.5.0:  nil   C stack overflow
--                   stack traceback:
--                       [C]: in global 'load'
--                       ...: in main chunk
--                       [C]: in ?

-- Under pcall, even reference returns the clean message (no traceback):
print((pcall(load, deep, "=t")))
--> both:      true   (load returns nil + "C stack overflow" as values)
