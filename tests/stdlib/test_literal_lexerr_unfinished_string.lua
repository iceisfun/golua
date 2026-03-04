-- Test: Lexer errors for unfinished strings
-- From: literals.lua
-- What: Tests that the lexer produces proper EOF error messages for various forms of unfinished strings and long strings.

do
  local function lexerror (s, err)
    local st, msg = load('return ' .. s, '')
    if err ~= '<eof>' then err = err .. "'" end
    assert(not st and string.find(msg, "near .-" .. err))
  end

  lexerror("[=[alo]]", "<eof>")
  lexerror("[=[alo]=", "<eof>")
  lexerror("[=[alo]", "<eof>")
  lexerror("'alo", "<eof>")
  lexerror("'alo \\z  \n\n", "<eof>")
  lexerror("'alo \\z", "<eof>")
  lexerror([['alo \98]], "<eof>")
end
