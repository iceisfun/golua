-- string.format hex integer and hex float zero-padding should match C/Lua 5.4 behavior.
--
-- Issues:
-- 1. %#020x: Go excludes "0x" from width count; Lua includes it
-- 2. %020a: Zero padding should go after "0x" prefix, not before
-- 3. Subnormal floats: should use denormalized mantissa form (exponent -1022)

-- Test 1: %#020x — "0x" prefix counts toward width
assert(string.format("%#020x", 255) == "0x0000000000000000ff",
  "got: " .. string.format("%#020x", 255))
assert(string.format("%#020X", 255) == "0X0000000000000000FF",
  "got: " .. string.format("%#020X", 255))

-- Test 2: %020x without # — plain zero padding
assert(string.format("%020x", 255) == "000000000000000000ff",
  "got: " .. string.format("%020x", 255))

-- Test 3: %#020x with zero value — # has no effect for zero
assert(string.format("%#020x", 0) == "00000000000000000000",
  "got: " .. string.format("%#020x", 0))

-- Test 4: %020a — zero padding after "0x" prefix
assert(string.format("%020a", 1.5) == "0x0000000000001.8p+0",
  "got: " .. string.format("%020a", 1.5))
assert(string.format("%020a", -1.5) == "-0x000000000001.8p+0",
  "got: " .. string.format("%020a", -1.5))

-- Test 5: subnormal float formatting
assert(string.format("%a", 5e-324) == "0x0.0000000000001p-1022",
  "got: " .. string.format("%a", 5e-324))

-- Test 6: normal floats still work
assert(string.format("%.1a", 1.5) == "0x1.8p+0",
  "got: " .. string.format("%.1a", 1.5))
assert(string.format("%.0a", 1.0) == "0x1p+0",
  "got: " .. string.format("%.0a", 1.0))

print("PASS")
