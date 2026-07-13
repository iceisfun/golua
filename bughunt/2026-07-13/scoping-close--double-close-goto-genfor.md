# goto-continue inside generic for prematurely closes the loop's closing value

**Verification: CONFIRMED (2026-07-13, adversarial re-run vs /usr/bin/lua5.5.0).**

Forward `goto` to an end-of-body label (`::cont::` as the last statement — the
standard goto-continue idiom) inside a generic `for` emits an OP_CLOSE whose
level is patched too LOW, closing the loop's to-be-closed closing value (4th
value of the iterator explist) while the loop is still running. With
`io.lines` this kills the open file mid-iteration.

## Minimized repro (verified)
```lua
local log = {}
local tbc = setmetatable({}, {__close=function() log[#log+1]="CLOSE" end})
local n = 0
for i in function() n=n+1 if n<=2 then return n end end, nil, nil, tbc do
  log[#log+1] = "iter"..i
  goto cont
  ::cont::
end
print(table.concat(log, ","))
```
golua:   `iter1,CLOSE,iter2`   (closing value closed during iteration 1)
lua5.5:  `iter1,iter2,CLOSE`

## Practical repro (io.lines) — verified
```lua
-- lines.txt contains: one\ntwo\nthree\n
local ok, err = pcall(function()
  for line in io.lines("lines.txt") do
    goto cont
    ::cont::
  end
end)
print(ok, err)
```
golua:   `false   <file>:4: attempt to use a closed file`  (fails entering iteration 2)
lua5.5:  `true    nil`

## Boundary conditions (all verified on both interpreters)
- Triggers ONLY when the label is the last statement of the generic-for body
  itself (the `atBlockEnd` path in `compileLabelStmt`). A label followed by
  any other statement does NOT diverge.
- Wrapping the goto/label in ANY inner scope inside the loop body — a bare
  `do ... end` block or an inner while/numeric-for — does NOT diverge
  (inner constructs open their own compiler scope).
- Backward gotos and `break` are unaffected.
- Reproduces with or without named locals declared before the loop.
- Deterministic `<close>`/TBC semantics — NOT GC/finalizer-timing dependent;
  not in wontfix/.

## Root cause (compiler/compile_stmt.go + compile_control.go)
`compileForInStmt` (compile_control.go:495) opens exactly ONE scope for the
entire for statement; the loop body's `compileBlock` does not open its own.
The hidden control locals — including the closing variable at base+2, marked
`attribClose` (compile_control.go:578) — and the user loop variables all live
in that single scope.

`compileGotoStmt` (forward-goto branch, compile_stmt.go:1135-1149) emits a
placeholder `OP_CLOSE regTop()` because `needsClose(0)` is always true inside
a generic for (the hidden closing slot is attribClose). When the label
resolves at block end, `compileLabelStmt` sets
`labelNLocals = scope.nLocals` (compile_stmt.go:1193-1195) — but `scope` is
the for statement's own scope, so `scope.nLocals` is the active-local count
from BEFORE the `for`. That makes `labelNLocals < pg.nLocals`, and the
placeholder is patched to `OP_CLOSE regBaseForLocals(labelNLocals)`
(compile_stmt.go:1229-1231) = the for-state base register — at or below the
closing variable at base+2. Executing the goto then closes the loop's TBC
value mid-loop.

Reference Lua 5.5 `forbody()` (lparser.c) does
`enterblock(fs, &bl, 0)` for the loop body AFTER `adjustlocalvars` of the 3
internal "(for state)" vars, so a block-end label there resolves at a level
that still includes the control/closing registers and no CLOSE below them is
ever emitted. golua needs the equivalent: the block-end label level inside a
generic-for body must not drop below the hidden for-state locals (i.e. the
body needs its own scope, or the atBlockEnd level must floor at the for-state
locals).

Fuzz seed 152 (gen.py) found a feedback variant where the spurious early
close changes `#LOG`, causing an extra `local v <close>` declaration+close
(observable double-close pattern) — see scratch mismatch_152.lua.

Severity: wrong-result / wrong-error-behavior (resource closed
mid-iteration; io.lines loops with goto-continue break outright).
