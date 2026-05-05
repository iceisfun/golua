-- broken_fuzz_utf8_codes_orphan_lookahead:
-- utf8.codes iterator emits a valid codepoint when the byte immediately
-- after it is an orphan continuation. Reference Lua (5.4 and 5.5) check
-- `iscontp(next)` after a successful decode and error BEFORE yielding the
-- pair. golua emits the pair, then errors on the next iteration.
--
-- BROKEN: stdlib/utf8.go iter_aux around lines 346-353 lacks the
-- post-decode lookahead at `s[next]` that reference Lua uses to detect
-- orphan continuation bytes following the just-decoded character.
--
-- For s = "F\x1F\xBC\x05":
--   pos 0: 'F' (1B valid). next=1, s[1]='\x1F' (not continuation) -> emit (1,70)
--   pos 1: '\x1F' (1B valid). next=2, s[2]='\xBC' IS continuation
--     -> 5.4/5.5 error "invalid UTF-8 code" before emitting (2, 31)
--     -> golua emits (2, 31), then errors at pos 2.
--
-- Affects both lax and strict modes.
--
-- Reference (lua5.5.0 and lua 5.4.8 — same output):
--   yields collected: ["1:70"]
--   error: "invalid UTF-8 code"
--
-- golua today:
--   yields collected: ["1:70", "2:31"]   <- extra emit
--   error: "invalid UTF-8 code"
--
-- Discovered: differential fuzz 2026-05-04 (utf8 wave-2 agent).

local s = "F\x1F\xBC\x05"
local out = {}
local ok, err = pcall(function()
  for p, c in utf8.codes(s) do
    table.insert(out, p .. ":" .. c)
  end
end)

assert(ok == false, "iteration should error")
assert(#out == 1, "iterator should emit exactly 1 codepoint before erroring; got " ..
  #out .. " (" .. table.concat(out, ",") .. ")")
assert(out[1] == "1:70", "first emit should be 1:70 ('F'); got " .. tostring(out[1]))

print("ok")
