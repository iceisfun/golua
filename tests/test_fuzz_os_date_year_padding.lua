-- broken_fuzz_os_date_year_padding:
-- os.date("!%Y", ...) zero-pads to 4 digits for years < 1000. Reference
-- Lua (delegating to glibc's strftime) emits the natural width.
--
-- BROKEN: vm/default_os.go around lines 244-245 and 316-317 uses Go's
--   t.Format("2006")  /  fmt.Sprintf("%04d", ...)
-- which always zero-pads to 4 columns. Reference glibc emits a variable-
-- width number for %Y. The same defect applies to %G (ISO week-based
-- year) and cascades through compound specs %c, %F, %D, %x for years
-- below 1000.
--
-- Reference (lua5.5.0 and lua 5.4.8 — same), under TZ=UTC LC_ALL=C:
--   os.date("!%Y", os.time({year=1, month=1, day=1, hour=0, min=0, sec=0}))
--     -> "1"
--   os.date("!%Y", os.time({year=99,  ...})) -> "99"
--   os.date("!%Y", os.time({year=999, ...})) -> "999"
--
-- golua today:
--   "0001", "0099", "0999"
--
-- Discovered: differential fuzz 2026-05-04 (os wave-3 agent).

local function y(year)
  return os.date("!%Y", os.time({year=year, month=1, day=1, hour=0, min=0, sec=0}))
end

assert(y(1)   == "1",   "%Y for year=1 should be '1', got " .. tostring(y(1)))
assert(y(99)  == "99",  "%Y for year=99 should be '99', got " .. tostring(y(99)))
assert(y(999) == "999", "%Y for year=999 should be '999', got " .. tostring(y(999)))
assert(y(2024) == "2024", "%Y for year=2024 should be '2024'")

print("ok")
