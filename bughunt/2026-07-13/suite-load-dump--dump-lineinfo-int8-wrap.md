# wrong-result: string.dump/load corrupts line info when a line delta exceeds 127 (int8 wrap → negative error lines)

After a string.dump → load round trip, golua reports wrong (negative) line
numbers in errors/tracebacks whenever consecutive instructions are more than
127 source lines apart. The per-instruction line delta is evidently stored as a
signed byte with no absolute-line-info escape (reference emits ABSLINEINFO
entries when a delta doesn't fit), so a gap of 128 wraps: line 129 becomes
-127. golua's own dump loaded by golua's own load — no reference bytecode
involved. Direct (non-dumped) execution is correct; linedefined/lastlinedefined
survive fine; only the per-instruction line table wraps.

## Minimized repro (verified 2026-07-13)
```lua
local f = load("local function f()" .. string.rep("\n", 128) .. "error('x') end return f", "@c")()
print(select(2, pcall(load(string.dump(f)))))
```
golua: `c:-127: x` — lua5.5.0: `c:129: x`

## Full repro (threshold sweep)
```lua
for _, gap in ipairs{126, 127, 128, 129} do
  local src = "local function f()" .. string.rep("\n", gap) .. "error('x') end return f"
  local f = assert(load(src, "@c"))()
  local g = assert(load(string.dump(f)))
  print(gap, select(2, pcall(f)), select(2, pcall(g)))
end
```

## golua
```
126	c:127: x	c:127: x
127	c:128: x	c:128: x
128	c:129: x	c:-127: x
129	c:130: x	c:-126: x
```

## lua5.5.0
```
126	c:127: x	c:127: x
127	c:128: x	c:128: x
128	c:129: x	c:129: x
129	c:130: x	c:130: x
```

Why wrong: error line numbers (and debug.getinfo currentline) after loading a
dumped chunk must match the original; 128+ line gaps are common in real files
(license headers, generated code), so precompiled chunks get garbage line
numbers. Threshold is exactly delta > 127 (int8).

## Additional manifestation: line hooks skip/misreport lines
For a loop body separated from its `for` header by >127 lines, a "l" hook on
the round-tripped function fires `9,2,3,10` in golua vs
`9,2,3,143,3,143,3,143,3,145,10` in reference (loop-body lines vanish), i.e.
debuggers/coverage tools see wrong coverage on dumped chunks:
```lua
local src = "local function f()\nlocal s = 0\nfor i = 1, 3 do"
  .. string.rep("\n", 140) .. "s = s + i\nend\nreturn s\nend\nreturn f"
local g = assert(load(string.dump(assert(load(src, "@nd"))())))
debug.sethook(function(e,l) io.write(l, ",") end, "l"); g(); debug.sethook()
```

## Mechanism (confirmed in source)
- stdlib/string_dump.go:226 `d.writeByte(byte(int8(line - prev)))` — silently
  truncates any delta outside [-128,127]; line 231 writes abslineinfo count 0
  ("empty for simplicity"), so there is no escape entry.
- compiler/undump.go:308 reads `int(int8(u.readByte()))` and accumulates, and
  undump.go:315-319 reads but DISCARDS abslineinfo entries — so even a
  reference-style fix on the dump side needs undump to honor abslineinfo (and
  the reference convention is a delta sentinel of -128 pointing into the
  absolute table).

## Verification: CONFIRMED (2026-07-13)
Independently reproduced by adversarial verifier: both the sweep and the
2-line minimized repro diverge exactly as claimed (golua binary vs
/usr/bin/lua5.5.0, both under ulimit/timeout). Threshold is exactly
delta > 127. Not error-message prose — the reported *line number* is wrong
(negative), affecting errors, tracebacks, debug.getinfo, and line hooks on
any dump/load round trip. Not GC/timing dependent. Not covered by wontfix/
(untrusted-binary-chunks is about crafted malicious chunks; this is golua's
own dump loaded by golua's own load). Reference behavior confirmed in
lua-5.5.0 ldump.c/lobject.h: deltas that don't fit a signed byte get an
ABSLINEINFO (-0x80) sentinel plus an abslineinfo entry, which golua neither
emits (string_dump.go:226,231) nor honors on load (undump.go:308,315-319).
