-- Test: Unfinished string terminated by newline should show partial content in "near"
-- Bug: GoLua shows <eof> instead of the partial string content.
-- Lua 5.4: unfinished string near ''hello'
-- GoLua:   unfinished string near <eof>

local _, err = load("x = 'hello\nworld'")
assert(err:find("near ''hello'"), "single-quoted: got: " .. tostring(err))

_, err = load('x = "hello\nworld"')
assert(err:find('near \'"hello\''), "double-quoted: got: " .. tostring(err))

print("PASS")
