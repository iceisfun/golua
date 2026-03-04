-- Test: Lexer errors for invalid hex escape sequences
-- From: literals.lua
-- What: Tests that the lexer produces proper error messages for malformed \x escape sequences in strings.

do
  local function lexerror (s, err)
    local st, msg = load('return ' .. s, '')
    if err ~= '<eof>' then err = err .. "'" end
    assert(not st and string.find(msg, "near .-" .. err))
  end

  lexerror([["abc\x"]], [[\x"]])
  lexerror([["abc\x]], [[\x]])
  lexerror([["\x]], [[\x]])
  lexerror([["\x5"]], [[\x5"]])
  lexerror([["\x5]], [[\x5]])
  lexerror([["\xr"]], [[\xr]])
  lexerror([["\xr]], [[\xr]])
  lexerror([["\x.]], [[\x.]])
  lexerror([["\x8%"]], [[\x8%%]])
  lexerror([["\xAG]], [[\xAG]])
  lexerror([["\g"]], [[\g]])
  lexerror([["\g]], [[\g]])
  lexerror([["\."]], [[\%.]])
end
