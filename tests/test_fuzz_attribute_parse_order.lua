-- test_fuzz_attribute_parse_order:
-- A malformed local-variable attribute must report the missing '>' BEFORE
-- validating the attribute name. Lua 5.5's getvarattribute does
-- str_checkname -> checknext(ls,'>') -> THEN strcmp. So `local x<weird>=5`
-- (the lexer reads `>=` as one token) reports the '>' error, not the
-- unknown-attribute error.
--
-- Discovered: differential scout 2026-05-20 (control-flow agent).

local err = select(2, load("local x<weird>=5"))
assert(err:find("'>' expected", 1, true),
  "expected \"'>' expected\" error, got: " .. tostring(err))
assert(not err:find("unknown attribute", 1, true),
  "must not validate the attribute name first, got: " .. tostring(err))

-- A well-formed but unknown attribute (with a real '>') still reports it.
local err2 = select(2, load("local y <weird> = 5"))
assert(err2:find("unknown attribute 'weird'", 1, true),
  "expected unknown-attribute error, got: " .. tostring(err2))

-- Valid attributes still parse.
assert(load("local z <const> = 5"), "<const> must still parse")

print("ok")
