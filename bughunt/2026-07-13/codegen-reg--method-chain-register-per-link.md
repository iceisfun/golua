# Method-call chains consume 2 registers per link — valid chains >= 128 links rejected with "too many registers"

Severity: wrong-error-behavior (valid programs rejected; reference runs them in O(1) registers)

golua compiles a postfix chain `o:m():m():m()...` inside-out: the disassembly of a
3-link chain shows registers allocated top-down (SELF 5, CALL, MOVE 4, SELF 3, CALL,
MOVE 2, SELF 1 — "7 slots"), i.e. **2 registers per chain link held simultaneously**.
Reference Lua compiles the same chain iteratively, reusing one register pair, so
chain length is unlimited. golua rejects any method chain of >= 128 links (fewer if
locals are in scope) with `too many registers (limit is 255)`.

The same leak affects `.field()` chains: `t.a().a().a()...` (300 links) is also
rejected, while reference runs it.

Plain value-call chains `f()()()...` are fine (that shape was fixed in an earlier
campaign); only chains that re-index the intermediate result leak.

## Repro

```lua
-- 128-link method chain: python3 -c "print('local o = setmetatable({n=0}, {__index=function(t,k) return function(self) self.n=self.n+1 return self end end})'); print('local r = o' + ':m()'*128); print('print(r.n)')"
local o = setmetatable({n=0}, {__index=function(t,k) return function(self) self.n=self.n+1 return self end end})
local r = o:m():m() --[[ ... 128 times total ... ]]
print(r.n)
```

## golua output

```
golua: chain.lua: too many registers (limit is 255)
```

(verified: 124–127 links compile and run correctly, printing the link count;
exactly 128 links fail; adding 10 live locals makes even 124 links fail, so the
threshold drops with locals in scope)

## lua5.5.0 output

```
128
```

(works at 3000+ links; constant register usage: `SELF 1 1; CALL 1 2 2; SELF 1 1; ...`)

## Why it's wrong

Reference lparser compiles postfix suffixes left-to-right, discharging each call's
result into the chain's base register before compiling the next suffix, so register
use is constant. golua's chain compilation reserves the destination for the
outermost link first and recurses inward (visible as descending register numbers in
`go run ./cmd/luac` output: `MOVE 6 0 / SELF 5 6 / CALL 5 / MOVE 4 5 / SELF 3 4 ...`),
holding 2 registers per link for the whole chain. Long builder-pattern or generated
chains that reference Lua accepts fail to compile.

Disassembly evidence (3-link chain): golua `cmd/luac` shows "7 slots" with
descending SELF registers; `luac5.5.0 -l` shows "4 slots" with `SELF 1 0 /
CALL 1 2 2 / SELF 1 1 / CALL 1 2 2 / SELF 1 1 / CALL 1 2 2` — constant reuse.

Found by long-chain shape sweep, 2026-07-13 codegen-reg lens.

## Verification: CONFIRMED (2026-07-13)

Independently reproduced with a freshly generated repro:

- 128-link `o:m()` chain: golua fails at compile time with
  `golua: chain128.lua: too many registers (limit is 255)`; lua5.5.0 prints `128`.
- Threshold sweep: 124/125/126/127 links all run correctly on golua (print
  124/125/126/127); 128 fails. 2 registers/link × 128 = 256 > 255 matches the
  per-link-leak model exactly.
- lua5.5.0 runs a 3000-link chain fine (`3000`) — reference register use is O(1),
  confirmed by `luac5.5.0 -l` on a 3-link chain: 3 slots, `SELF 1 0 / CALL 1 2 2 /
  SELF 1 1 / CALL 1 2 2 / SELF 1 1 / CALL 1 2 2` (register 1 reused every link).
- `.field()` chain (300 × `.a()`): golua rejects with
  `too many registers (limit is 255) in main function near 'a'`; lua5.5.0 runs it.
- Locals lower the threshold: with 10 locals in scope, a 124-link chain (which
  passes bare) also fails.
- Plain `f()()()...` 300-deep call chains compile fine on golua (only re-indexing
  chains leak).

Not a wontfix item (checked wontfix/README.md index — the compiler-limit entries
there cover error *wording*, not rejecting valid programs), not GC-related, and
not error-message-only: golua rejects at compile time a program reference Lua
5.5.0 compiles and runs. Genuine codegen deficiency in chain suffix compilation.
