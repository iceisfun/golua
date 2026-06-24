-- math.* functions coerce string arguments integer-first (luaO_str2num), so an
-- integer-parseable string like "-0" becomes the integer 0 -> +0.0, not the
-- float -0.0 a bare ParseFloat would produce. Matches reference Lua 5.5.
-- (Same class as the string.format "-0" integer-first fix.)

print(string.format("%.17g", math.sqrt("-0")))
--> =0

print(string.format("%.17g", math.sin("-0")))
--> =0

print(string.format("%.17g", math.fmod("-0", 1)))
--> =0

-- Whitespace-trimmed integer string still coerces integer-first.
print(string.format("%.17g", math.sqrt("  -0  ")))
--> =0

-- A genuine float string keeps its signed zero (not integer-parseable).
print(string.format("%.17g", math.sqrt("-0.0")))
--> =-0

-- Ordinary integer-string coercion is unaffected.
print(math.floor("10") + math.sqrt("16"))
--> =14.0
