-- Test: Lexer errors for invalid decimal escape sequences
-- From: literals.lua
-- What: Tests that the lexer produces proper error messages for out-of-range decimal escape sequences (>255).

do
  local function lexerror (s, err)
    local st, msg = load('return ' .. s, '')
    if err ~= '<eof>' then err = err .. "'" end
    assert(not st and string.find(msg, "near .-" .. err))
  end

  lexerror([["\999"]], [[\999"]])
  lexerror([["xyz\300"]], [[\300"]])
  lexerror([["   \256"]], [[\256"]])
end
