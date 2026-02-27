-- Bug: _VERSION is "Lua 5.5" instead of "Lua 5.4".
-- golua implements Lua 5.4, not 5.5.

assert(_VERSION == "Lua 5.4",
  "_VERSION should be 'Lua 5.4', got '" .. _VERSION .. "'")

print("PASS")
