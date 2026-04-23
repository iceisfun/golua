-- Test: Decimal escape too large (class of positions) — fuzz regression
-- What: Regression test for a lexer bug where a decimal escape > 255 was
-- reported as "unfinished string" whenever the offending literal was NOT
-- the first token on the line. The root cause was that the escape scanner
-- returned early and left the lexer mid-string; subsequent token requests
-- then saw the tail of the literal as an unfinished string.
--
-- The fix ensures the lexer consumes through the closing delimiter before
-- raising the "decimal escape too large" diagnostic, so the near-context
-- covers the full literal and the lexer state is clean afterwards.

do
  local function lexerror(s, err)
    local st, msg = load('return ' .. s, '')
    if err ~= '<eof>' then err = err .. "'" end
    assert(not st, "expected lexer to reject: " .. s)
    assert(string.find(msg, "decimal escape too large"),
      "expected 'decimal escape too large' in: " .. tostring(msg))
    assert(string.find(msg, "near .-" .. err),
      "expected near '" .. err .. "' in: " .. tostring(msg))
  end

  -- Offending literal as first / only token (already worked pre-fix).
  lexerror([["\261"]], [[\261"]])
  lexerror([["\999"]], [[\999"]])
  lexerror([["\300"]], [[\300"]])

  -- Offending literal NOT first on line (previously mis-reported as
  -- "unfinished string"). These are the regression cases.
  do
    local st, msg = load('x("a", "\\261")', '')
    assert(not st)
    assert(string.find(msg, "decimal escape too large"),
      "multi-arg call: " .. tostring(msg))
    assert(string.find(msg, [[near '"\261"']]),
      "multi-arg call near-context: " .. tostring(msg))
  end

  do
    local st, msg = load('local a = 1; local b = "\\256"', '')
    assert(not st)
    assert(string.find(msg, "decimal escape too large"),
      "after local: " .. tostring(msg))
  end

  do
    local st, msg = load('local x = "\\300" .. "\\400"', '')
    assert(not st)
    -- First offending literal is "\300".
    assert(string.find(msg, "decimal escape too large"),
      "concat chain: " .. tostring(msg))
    assert(string.find(msg, [[near '"\300"']]),
      "concat chain near-context: " .. tostring(msg))
  end

  -- EOF without closing quote, decimal escape first: near excludes the
  -- missing closing quote. Matches Lua 5.5.
  do
    local st, msg = load('x("\\261', '')
    assert(not st)
    assert(string.find(msg, "decimal escape too large"),
      "eof unterminated: " .. tostring(msg))
    assert(string.find(msg, [[near '"\261']]),
      "eof unterminated near-context: " .. tostring(msg))
  end

  -- Long strings do NOT interpret escapes, so [[\261]] is a valid literal.
  do
    local ok, val = pcall(load('return [[\\261]]', ''))
    -- load() succeeds; calling the chunk returns the literal text.
    local chunk = load('return [[\\261]]', '')
    assert(chunk, "long string with \\261 should parse")
    assert(chunk() == [[\261]])
  end

  -- Truly unfinished short string (no escapes) still reports
  -- "unfinished string", not "decimal escape too large".
  do
    local st, msg = load('x("oops', '')
    assert(not st)
    assert(string.find(msg, "unfinished string"),
      "unfinished no-escape: " .. tostring(msg))
  end
end

print("OK")
