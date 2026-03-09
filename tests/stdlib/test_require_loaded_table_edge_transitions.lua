-- require/package.loaded edge transitions and sparse searcher behavior.

do
  local originalSearchers = package.searchers
  local originalLoadedAlias = package.loaded

  local function restore()
    package.searchers = originalSearchers
    package.loaded = originalLoadedAlias
  end

  -- Replacing package.loaded inside a loader should not affect require's
  -- internal loaded-table bookkeeping for this call.
  do
    local internalLoaded = package.loaded
    internalLoaded.mod_swap_loaded = nil
    package.searchers = {
      function()
        return function(name)
          package.loaded = {}
          package.loaded[name] = "replacement-value"
          return nil
        end, "searcher-data"
      end,
    }

    local result, extra = require("mod_swap_loaded")
    assert(result == true, "expected require to return true, got " .. tostring(result))
    assert(extra == "searcher-data", "expected loader data pass-through")
    assert(internalLoaded.mod_swap_loaded == true, "expected internal loaded table to be updated")
    assert(package.loaded.mod_swap_loaded == "replacement-value", "expected replacement alias to be unchanged by require bookkeeping")
  end

  -- package.loaded may be reassigned to nil/false and require should still work.
  do
    package.searchers = {
      function()
        return function()
          return nil
        end, "ld"
      end,
    }

    package.loaded = nil
    local a, b = require("mod_loaded_nil")
    assert(a == true and b == "ld", "expected require success with package.loaded=nil")

    package.loaded = false
    local c, d = require("mod_loaded_false_alias")
    assert(c == true and d == "ld", "expected require success with package.loaded=false")
  end

  -- Sparse searchers stop at the first nil index and preserve not-found format.
  do
    package.searchers = {
      [1] = function()
        return "first-miss"
      end,
      [3] = function()
        return function()
          return "should-not-run"
        end, "late"
      end,
    }

    local ok, err = pcall(require, "mod_sparse_stop")
    assert(not ok)
    assert(string.find(err, "module 'mod_sparse_stop' not found:", 1, true), err)
    assert(string.find(err, "\n\tfirst-miss", 1, true), err)
    assert(string.find(err, "should-not-run", 1, true) == nil, err)
  end

  restore()
end
