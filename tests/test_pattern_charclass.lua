-- Bug 1: %w includes underscore. Lua 5.4 %w = letters + digits only (isalnum).
-- Bug 2: %p uses unicode.IsPunct instead of C's ispunct. Missing: $ + < = > ^ ` { | } ~

-- %w should NOT match underscore
assert(not string.find("_", "^%w$"),
  "%w should not match underscore, but it does")

-- %w should match letters and digits
assert(string.find("a", "^%w$"), "%w should match 'a'")
assert(string.find("Z", "^%w$"), "%w should match 'Z'")
assert(string.find("5", "^%w$"), "%w should match '5'")

-- %W (complement) SHOULD match underscore
assert(string.find("_", "^%W$"),
  "%W should match underscore since _ is not alphanumeric")

-- %p should match ALL ASCII punctuation per C's ispunct()
-- ispunct matches: ! " # $ % & ' ( ) * + , - . / : ; < = > ? @ [ \ ] ^ _ ` { | } ~
local punct_chars = '!"#$%&\'()*+,-./:;<=>?@[\\]^_`{|}~'
for i = 1, #punct_chars do
  local c = punct_chars:sub(i, i)
  assert(string.find(c, "^%p$"),
    "%p should match '" .. c .. "' but does not")
end

-- %p should NOT match letters, digits, space, or control chars
assert(not string.find("a", "^%p$"), "%p should not match 'a'")
assert(not string.find("5", "^%p$"), "%p should not match '5'")
assert(not string.find(" ", "^%p$"), "%p should not match space")

-- Verify %P complement works
assert(string.find("a", "^%P$"), "%P should match 'a'")
assert(not string.find("!", "^%P$"), "%P should not match '!'")

print("PASS")
