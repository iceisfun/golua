# Nested table constructors >= 129 deep silently mis-compile (register index wraps past 255, no limit check)

Severity: wrong-result (SILENT miscompilation)

golua's `compileTableConstructor` consumes **2 registers per nesting level**
(reference Lua uses 1), and its manual `fs.freeReg = arrReg + 1` /
`fs.freeReg = reg + 1` bumps update `maxReg` WITHOUT the `MaxRegs` check that
`reserveReg()` performs. A 129-deep nested constructor therefore compiles to a
function with **258 slots** (see `go run ./cmd/luac`: "0 params, 258 slots"),
and the NEWTABLE/MOVE/SETLIST instructions targeting registers >= 256 silently
wrap modulo 256 in the 8-bit A/B fields. The chunk loads and runs without any
error but builds a corrupted structure. Reference Lua compiles the same chunk
correctly to at least depth 190; its parser recursion limit rejects somewhere
in 191..198 with "C stack overflow" (golua's own recursion limit rejects at
199 with the same message).

Corruption window (verified 2026-07-13): depth <= 128 correct on both;
depth 129..190 golua silently corrupts while reference is correct (golua keeps
silently corrupting through 198, past the point where reference has already
switched to a clean "C stack overflow" reject); depth >= 199 both reject.

## Repro (depth 129; any depth 129-199 corrupts the same way)

```lua
local t = {{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{{42}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}
local c, n = t, 0
while type(c) == "table" and c[1] ~= nil do c = c[1]; n = n + 1 end
print(n, c)
```

(generate with: `python3 -c "d=129; print('local t = ' + '{'*d + '42' + '}'*d)"`)

## golua output

```
127	table: 0xc0001f4320
```

(nesting stops at 127; the innermost reachable table is EMPTY — the `42` and
the two deepest tables are lost)

## lua5.5.0 output

```
129	42
```

## Why it's wrong

Reference `lparser.c` uses one register per constructor nesting level and
runs `luaK_checkstack` (lcode.c:476, `luaY_checklimit(fs, newstack,
MAX_FSTACK, "registers")`) on every reservation, erroring with a "too many
registers" limit error before ever emitting an operand that does not fit —
it can never silently wrap. golua's constructor path bypasses `reserveReg()`'s limit check,
so `maxReg`/`MaxStack` grows past 255 and the emitted 8-bit register operands
(`uint32(reg)<<PosA`) truncate. Two independent defects:

1. Manual freeReg bumps in `compiler/compile_expr.go` `compileTableConstructor`
   (both the `fs.freeReg <= reg` prologue and the `arrReg >= fs.freeReg` array
   branch) skip the `MaxRegs` limit enforcement.
2. The layout costs 2 registers/level vs reference's 1, halving the usable
   nesting depth even once the limit check is fixed (reference handles 199,
   a checked golua would reject at ~128).

## Related manifestation: valid hash-nested constructors falsely rejected

The 2-regs-per-level over-consumption also makes golua REJECT programs
reference accepts, where the path goes through the checked `reserveReg()`:

```lua
-- python3 -c "d=150; print('local t = ' + '{k='*d + '1' + '}'*d)"
local t = {k={k={k= ... 150 deep ... 1}}}
print("compiled")
```

golua: `too many registers (limit is 255)` (compile error)
lua5.5.0: compiles and runs fine (also fine at depth 190).

Likewise `{ {k=` mixed nesting at depth 90 is rejected by golua, accepted by
reference.

## Realistic shallow variant (150 locals + 60-deep constructor)

The wrap point scales down with live locals — with 150 locals in scope a
60-deep constructor already crosses 255 (150 + 2*60 = 270):

```lua
-- local v0 = 0 ... local v149 = 149   (150 locals)
-- local t = {{{...60 deep...42...}}}
-- walk t
```

golua: `nestmix.lua:151: attempt to index a non-table value` (runtime error at
the constructor line — SETLIST's wrapped A operand hits a non-table register).
lua5.5.0: `60	42	0	149` (correct).

Found by deep-nesting boundary sweep, 2026-07-13 codegen-reg lens.

## Verification: CONFIRMED (2026-07-13, adversarial re-run)

Independently regenerated and re-ran all three variants against
`/usr/bin/lua5.5.0` under `ulimit -v 16GB` + `timeout 15`:

- Depth sweep on the primary repro: 126/127/128 identical (`N 42`) on both;
  **129 -> golua `127	table: 0x...` vs oracle `129	42`**; 130 and 190 corrupt
  the same way; 198 -> golua still silently corrupt while oracle rejects with
  "C stack overflow"; 199 -> both reject "C stack overflow".
- `go run ./cmd/luac` on the depth-129 chunk confirms
  `0 params, 258 slots` — with `SizeA = 8` (instruction.go: `PosA = 7`,
  `PosK = 15`), any register operand >= 256 truncates, spilling into the k bit.
  Mechanism confirmed in source: `compileTableConstructor`
  (compiler/compile_expr.go:1326-1331 prologue and :1394-1399 array branch)
  bumps `fs.freeReg`/`fs.maxReg` manually without `reserveReg()`'s
  `MaxRegs` check (compiler/compiler.go:547); the 2-regs-per-level cost comes
  from the temp+MOVE path at compile_expr.go:186-190.
- 150-locals + 60-deep variant: golua
  `attempt to index a non-table value` (runtime) vs oracle `60	42`. Reproduced.
- `{k=` 150-deep false-reject: golua `too many registers (limit is 255)`
  (compile error) vs oracle runs fine (`150	42`). Reproduced.

Not a wontfix (silent wrong-result, not error-message wording — the
`load-stack-overflow-traceback` wontfix covers only limit-error *prose*);
no GC involvement. Reference semantics are unambiguous: lcode.c
`luaK_checkstack` guards every reservation, so reference either runs the
program correctly or rejects it cleanly — it never emits wrapped operands.
Severity "wrong-result / silent miscompilation" is accurate.
