-- Test os.execute("") returns "exit" status
-- os.execute("") should return true, "exit", 0 (shell runs empty command successfully)

local a, b, c = os.execute("")
assert(a == true, "expected true, got " .. tostring(a))
assert(b == "exit", "expected 'exit', got " .. tostring(b))
assert(c == 0, "expected 0, got " .. tostring(c))

print("PASS")
