-- require/searcher error semantics parity with Lua 5.4.

do
  local original = package.searchers

  -- Searcher errors must propagate immediately.
  package.searchers = {
    function()
      error("searcher exploded")
    end,
    function()
      return "should not be reached"
    end,
  }
  local ok, err = pcall(require, "mod_searcher_error")
  assert(not ok)
  assert(string.find(err, "searcher exploded", 1, true), err)
  assert(string.find(err, "not found", 1, true) == nil, err)

  -- String-returning searchers must appear as separate indented lines.
  package.searchers = {
    function()
      return "first miss"
    end,
    function()
      return "second miss"
    end,
  }
  ok, err = pcall(require, "mod_missing")
  assert(not ok)
  assert(string.find(err, "module 'mod_missing' not found", 1, true), err)
  assert(string.find(err, "\n\tfirst miss", 1, true), err)
  assert(string.find(err, "\n\tsecond miss", 1, true), err)

  -- Numeric-returning searchers are also appended (stringified), while
  -- other non-function/non-string values are ignored.
  package.searchers = {
    function()
      return true
    end,
    function()
      return 123
    end,
    function()
      return {}
    end,
  }
  ok, err = pcall(require, "mod_numeric_miss")
  assert(not ok)
  assert(string.find(err, "module 'mod_numeric_miss' not found", 1, true), err)
  assert(string.find(err, "\n\t123", 1, true), err)
  assert(string.find(err, "true", 1, true) == nil, err)

  package.searchers = original
end
