-- demo.lua: Demonstrates glob matching from Lua
--
-- The Go host exposes a "glob" table with:
--   glob.match(pattern, name) -> boolean
--   glob.match_words(pattern, name) -> boolean
--   glob.match_named(pattern, text) -> boolean, table
--   glob.has_pattern(s) -> boolean

-- Basic pattern matching
print("--- Basic Matching ---")
print("Match 'hel*' vs 'hello':", glob.match("hel*", "hello"))
print("Match 'h?llo' vs 'hello':", glob.match("h?llo", "hello"))
print("Match 'h[ae]llo' vs 'hello':", glob.match("h[ae]llo", "hello"))
print("Match 'world' vs 'hello':", glob.match("world", "hello"))

-- Case-insensitive matching
print("\n--- Case Insensitive ---")
print("Match 'HELLO' vs 'hello':", glob.match("HELLO", "hello"))
print("Match 'hello' vs 'HELLO':", glob.match("hello", "HELLO"))

-- Word-based matching
print("\n--- Word Matching ---")
local labels = {
    "ORGANIC PEACH",
    "ORGANIC WHITE PEACH",
    "CONVENTIONAL PEACH",
    "ORGANIC APPLE",
}
local pattern = "ORG* PEACH"
print("Pattern: " .. pattern)
for _, label in ipairs(labels) do
    local ok = glob.match_words(pattern, label)
    print("  " .. label .. " -> " .. tostring(ok))
end

-- Named captures
print("\n--- Named Captures ---")
local routes = {
    "/api/v1/users",
    "/api/v2/orders",
    "/api/v1/products",
    "/web/home",
}
local route_pattern = "/api/:version/:resource"
print("Route pattern: " .. route_pattern)
for _, route in ipairs(routes) do
    local ok, caps = glob.match_named(route_pattern, route)
    if ok then
        print("  " .. route .. " -> version=" .. caps.version .. ", resource=" .. caps.resource)
    else
        print("  " .. route .. " -> no match")
    end
end

-- Filtering with the helper script
print("\n--- Filtering (via dofile) ---")
local filter = dofile("filter.lua")

local items = {"alpha-100", "alpha-200", "beta-100", "beta-300", "gamma-100"}
local results = filter.select(items, "alpha-*")
print("Items matching 'alpha-*':")
for _, item in ipairs(results) do
    print("  " .. item)
end

local results2 = filter.select(items, "*-100")
print("Items matching '*-100':")
for _, item in ipairs(results2) do
    print("  " .. item)
end

-- Pattern detection
print("\n--- Pattern Detection ---")
local inputs = {"hello", "h*llo", "config.json", "*.lua", "[test]"}
for _, s in ipairs(inputs) do
    print("  " .. s .. " has patterns: " .. tostring(glob.has_pattern(s)))
end
