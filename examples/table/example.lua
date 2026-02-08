-- Demonstrate deterministic table iteration

-- Create a table with hash keys in a specific order
local config = {}
config["host"] = "localhost"
config["port"] = 8080
config["debug"] = true

-- First iteration: collect all keys
print("First iteration:")
local keys1 = {}
for k, v in pairs(config) do
    print("  " .. tostring(k) .. " = " .. tostring(v))
    keys1[#keys1 + 1] = k
end

-- Second iteration: should produce the same order
print("\nSecond iteration:")
local keys2 = {}
for k, v in pairs(config) do
    print("  " .. tostring(k) .. " = " .. tostring(v))
    keys2[#keys2 + 1] = k
end

-- Verify order is the same
print("\nOrder check:")
assert(#keys1 == #keys2, "key counts differ")
for i = 1, #keys1 do
    assert(keys1[i] == keys2[i], "key order differs at index " .. i)
end
print("  Both iterations produced identical key order")

-- Mixed array + hash table
print("\nMixed table (array + hash):")
local mixed = {10, 20, 30, name = "mixed", type = "demo"}
for k, v in pairs(mixed) do
    print("  [" .. tostring(k) .. "] = " .. tostring(v))
end
