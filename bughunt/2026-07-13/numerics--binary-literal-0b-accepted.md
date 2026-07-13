# golua lexer accepts non-standard binary literals `0b101`

Reference Lua 5.5 has no binary integer literals; `0b101` must be a compile
error ("malformed number"). golua's lexer parses it as binary and evaluates it.
A chunk that must fail to load in reference Lua loads and runs in golua.

## Repro

```lua
print(0b101)          -- golua: 5      lua5.5.0: compile error
local f, e = load("return 0b1111")
print(f and f(), e)   -- golua: 15 nil lua5.5.0: nil  malformed number near '0b1111'
```

## golua output

```
5
15      nil
```

## lua5.5.0 output

```
lua5.5.0: repro.lua:1: malformed number near '0b101'
```

Why wrong: language-level extension divergence — source accepted by golua is
rejected by every reference interpreter (and vice versa detection: feature
probes via `load` report the wrong dialect). `0b2`/`0b` are still rejected.

NOTE: this appears INTENTIONAL — `lexer/lexer_literal.go` has a dedicated
`scanBinaryNumber` with tests (`lexer/lexer_test.go`, `ast/ast_test.go`). If it
is a deliberate extension it belongs in wontfix/ or README so parity hunts stop
re-finding it; if not, it should be removed for 5.5 conformance.

## Verification: CONFIRMED (2026-07-13)

Independently reproduced against `/usr/bin/lua5.5.0`. Minimized repro (single
statement, no stdlib needed):

```lua
a = 0b1
```

- golua: exits 0, silently succeeds (`a == 1`).
- lua5.5.0: `malformed number near '0b1'`, exit 1.

`print(0b101)` prints `5` on golua; and `load("return 0b1111")` returns a
function on golua (`15  nil`) vs `nil  [string "return 0b1111"]:1: malformed
number near '0b1111'` on lua5.5.0 — so the divergence is visible both as a
hard compile failure and through catchable `load` (wrong dialect probe).

Oracle correctness checked: reference `llex.c read_numeral` only special-cases
`0x`/`0X` (`check_next2(ls, "xX")`); the 5.5 manual defines no binary numeral
form. Rejecting `0b101` is correct reference behavior.

Scope checks: not in `wontfix/` (README index has no binary-literal entry);
not GC/finalization-related; not error-message-only (accepted program vs
compile error).

Intentionality evidence: commit `ccb5f19` "feat: add support for binary
integer literals (0b) to lexer and compiler". The extension is real but
UNDOCUMENTED — no README/wontfix mention anywhere in the repo. Disposition is
a maintainer call: either remove for 5.5 conformance or add a `wontfix/`
entry (`binary-integer-literals`) documenting it as a deliberate extension.
