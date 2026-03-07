-- Test: heavy.lua - loadrepfunc helper
-- From: heavy.lua
-- What: Helper that loads a function with a dynamically generated body to test compiler limits.
--       Verifies that load() handles incremental reader functions for function bodies.

do
  local function loadrepfunc (prefix, f)
    local count = -1
    local function aux ()
      count = count + 1
      if count == 0 then
        return prefix
      else
        if count % (0x100000) == 0 and io and io.stderr then
          io.stderr:write("(", count // 2^20, " M)")
        end
        return f(count)
      end
    end
    local st, msg = load(aux, "k")
    msg = string.match(msg, "^[^\n]+")
  end

  -- Test: loading a function with many numeric constants
  loadrepfunc("function foo () return {0,",
      function (n)
        -- generate simple numeric constants until hitting a limit
        return tostring(n) .. ","
      end)
end
