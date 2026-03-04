-- Test: Lexer errors for invalid UTF-8 escape sequences
-- From: literals.lua
-- What: Tests that the lexer produces proper error messages for malformed \u{} escape sequences (too large, missing braces, no digits).

do
  local function lexerror (s, err)
    local st, msg = load('return ' .. s, '')
    if err ~= '<eof>' then err = err .. "'" end
    assert(not st and string.find(msg, "near .-" .. err))
  end

  lexerror([["abc\u{100000000}"]], [[abc\u{100000000]])   -- too large
  lexerror([["abc\u11r"]], [[abc\u1]])    -- missing '{'
  lexerror([["abc\u"]], [[abc\u"]])    -- missing '{'
  lexerror([["abc\u{11r"]], [[abc\u{11r]])    -- missing '}'
  lexerror([["abc\u{11"]], [[abc\u{11"]])    -- missing '}'
  lexerror([["abc\u{11]], [[abc\u{11]])    -- missing '}'
  lexerror([["abc\u{r"]], [[abc\u{r]])     -- no digits
end
