# Named vararg table is eagerly materialized; ref 5.5 keeps it virtual (debug API observable)

Reference 5.5 uses a *virtual* vararg table (lparser.c check_readonly /
PF_VATAB): unless the function needs a real table (t escapes, t[i]=v,
setmetatable(t,..), etc.), reads like `t.n`, `t[i]`, `...` compile to
virtual accesses of the hidden varargs and NO table is stored in the
local slot. golua always materializes a real table into the slot.

Two observable consequences:

## Repro A — debug.getlocal shows the slot value
```lua
local function dbg(...t)
  local n, v = debug.getlocal(1, 1)
  print(n, type(v))
end
dbg(10, 20)
```
golua:   `t table`
lua5.5:  `t nil`

## Repro B — debug.setlocal can hijack `...`/t reads in golua only
```lua
local function d(...t)
  debug.setlocal(1, 1, {n=1, "injected"})
  return t.n, t[1], (...)
end
print(pcall(d, "orig"))
```
golua:   `true 1 injected injected`
lua5.5:  `true 1 orig orig`

When the table IS forced real (e.g. `local u = t` first), both agree
(getlocal returns the table; setlocal replacement is visible in both).

## Verification: CONFIRMED (2026-07-13)

Both repros re-run against scratchpad golua build and /usr/bin/lua5.5.0;
outputs match the claim exactly. Minimized further:

Minimized A (getlocal):
```lua
local function f(...t)
  print(select(2, debug.getlocal(1, 1)))
end
f()
```
golua: `table: 0x...` — lua5.5.0: `nil`

Minimized B (setlocal, wrong result):
```lua
local function d(...t)
  debug.setlocal(1, 1, {"injected"})
  return t[1]
end
print(d("orig"))
```
golua: `injected` — lua5.5.0: `orig`

Reference semantics verified in source: lparser.c `setvararg` sets
`PF_VAHID` (hidden varargs) by default; only `needvatab` sites
(check_readonly VVARGIND write, lcode.c:809/1130 — t escaping as a
value etc.) set `PF_VATAB`, which clears `PF_VAHID` at close
(lcode.c:1933) and makes ldo.c:501 build the real table at entry.
Otherwise `t[i]`/`t.n` compile to direct hidden-vararg access and the
reserved slot stays nil — so getlocal reads nil and setlocal writes a
dead register.

golua instead materializes unconditionally at call entry:
vm/vm_exec.go:86-88 (`if proto.HasNamedVarArg { stack[base+VarArgReg] =
createVarArgTable(...) }`) and same in the tail-call path
(vm_exec.go:1355), with all `t` accesses routed through that register
(e.g. vm_exec.go:1941), so setlocal replaces the backing store for
`t[i]` and `...`.

Cross-checks: `local u = t` (needvatab-forced) agrees in both — getlocal
shows a table with correct `n`/elements; `t = 1` is rejected with the
identical "attempt to assign to const variable 't'" error in both. Not
in wontfix/ index; no GC/finalizer dependence; genuine value divergence,
not error-message prose. In-scope precedent: the 2026-06-06 for-loop
slot-layout work explicitly targeted debug.getlocal slot parity, and the
"(vararg table)" hidden-slot half of this was already fixed there
(HasVarArgSlot nil-ing); the named-vararg half is what remains.

Severity note: only observable through debug.getlocal/setlocal (plus a
per-call table allocation cost for read-only named varargs); no
divergence without the debug library.

## Why wrong
Observable divergence through the debug API for any vararg function that
only reads its varargs; also lets debuggers mutate what ref treats as
immutable virtual state. (Note: memory says the "5.5 (vararg table) slot"
was a known deferral from the for-loop layout project; this pins down the
exact observable semantics.)

Severity: wrong-result (debug API only).

## Disposition 2026-07-13: DEFERRED (documented, not fixed this round)

Matching reference requires the PF_VATAB machinery end to end: compiler
escape analysis over every use of the named vararg (only "t escapes /
t[i]=v / setmetatable(t,...)" forces a real table), virtual-access
compilation of t.n/t[i]/... against the hidden varargs, and entry/tail-call
changes in the VM. The divergence is observable only through
debug.getlocal/setlocal on functions that never force the table real —
no non-debug program can tell the difference.

Worth doing eventually for a different reason: the eager materialization
costs a table allocation per call of any named-vararg function even when
the varargs are only read — a real perf item for the backlog. Fixing the
debug observability falls out of that work for free. Until then this stays
a known divergence of the debug introspection surface.
