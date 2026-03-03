-- Test: heavy.lua - Too many constants (includes loadrepfunc helper)
-- From: heavy.lua
-- What: Tests that loading a function with enormous unique string constants
--       hits the constant table limit

do
  local function loadrepfunc (prefix, f)
    local count = -1
    local function aux ()
      count = count + 1
      if count == 0 then
        return prefix
      else
        if count % (0x100000) == 0 then
          io.stderr:write("(", count // 2^20, " M)")
        end
        return f(count)
      end
    end
    local st, msg = load(aux, "k")
    print("\nmemory: ", collectgarbage'count' * 1024)
    msg = string.match(msg, "^[^\n]+")
    print("expected error: ", msg)
  end

  print("loading function with too many constants")
  loadrepfunc("function foo () return {0,",
      function (n)
        return string.char(34,
          ((n // 128^0) & 127) + 128,
          ((n // 128^1) & 127) + 128,
          ((n // 128^2) & 127) + 128,
          ((n // 128^3) & 127) + 128,
          ((n // 128^4) & 127) + 128,
          34, 44)
      end)
end
