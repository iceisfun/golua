-- broken_fuzz_io_open_mode_multi_b:
-- io.open(path, "rbb") (and similar with multiple trailing 'b') is
-- rejected with "invalid mode" by golua. Reference Lua accepts multiple
-- trailing 'b' bytes per its l_checkmode rule:
--   r/w/a, optional '+', then ANY NUMBER of trailing 'b'.
--
-- BROKEN: stdlib/io.go around lines 544-550 strict-matches at most one
-- trailing 'b'. Should match the reference's relaxed rule.
--
-- Reference (lua5.5.0 and lua 5.4.8 — same):
--   io.open(path, "rbb")    -> ok
--   io.open(path, "r+bb")   -> ok
--   io.open(path, "rbbbb")  -> ok
--
-- golua today:
--   -> nil + "invalid mode" / pcall raises "(invalid mode 'rbb')"
--
-- Discovered: differential fuzz 2026-05-04 (io wave-3 agent).

local p = os.tmpname()
io.open(p, "w"):close()  -- create

for _, mode in ipairs({"rbb", "r+bb", "rbbbb"}) do
  local f, err = io.open(p, mode)
  if not f then
    os.remove(p)
    error("io.open(" .. p .. ", '" .. mode .. "') should succeed; got: " .. tostring(err))
  end
  f:close()
end

os.remove(p)
print("ok")
