-- _VERSION should report "Lua 5.5" for Lua 5.5 compatibility.

assert(_VERSION == "Lua 5.5",
  "_VERSION should be 'Lua 5.5', got '" .. _VERSION .. "'")

print("PASS")
