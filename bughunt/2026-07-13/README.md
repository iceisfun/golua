# Bug hunt 2026-07-13 — differential vs lua5.5.0 (master, b51a63b)

8-lens multi-agent hunt, every finding adversarially re-verified against `/usr/bin/lua5.5.0`
(minimized, checked against `wontfix/` and GC-scope). 30 raw findings → **17 CONFIRMED bugs**,
8 error-message-only notes, 5 rejected/out-of-scope. Each `.md` in this directory has the
minimized repro + both outputs + a verification verdict section.

## CONFIRMED — ranked by severity

### Tier 1: silent corruption / host crash
| # | Finding | File |
|---|---------|------|
| 1 | **Stale `vm.top` after open-call/vararg table constructor** (`{1,mr()}`, `{...}`): a later metamethod dispatch to a Lua closure — or even built-in string-arith coercion (`"21" % 3`) — places the callee frame over live caller registers, silently nil-ing/overwriting locals and loaded call args. 4-line repro; also corrupts OP_SELF receivers. Root cause: `vm/vm_exec.go` `call()` `base=vm.top` + missing `vm.top` restore after SETLIST-multi. | `codegen-reg--metamethod-call-clobbers-live-registers-after-open-call-constructor.md` |
| 2 | **`string.unpack` with ≥252 decoded values leaks a Go runtime index-out-of-range panic** to the host (reference succeeds). Sandbox-guarantee violation. Stack never grown, unlike `string.byte`. | `strings-patterns--unpack-many-results-go-panic.md` |
| 3 | **Nested table constructors ≥129 deep silently miscompile**: 2 registers/level (ref: 1) and `compileTableConstructor`'s manual `freeReg` bumps skip the MaxRegs check → 8-bit register operands wrap mod 256 → corrupted structures, no error. With 150 live locals, depth 60 already corrupts. Also falsely rejects depth-150 `{k={k=…}}` that reference accepts. | `codegen-reg--nested-constructor-129-deep-register-wrap.md` |
| 4 | **`string.dump`/`load` corrupts line info when consecutive instructions are >127 source lines apart**: dump writes bare int8 deltas with empty abslineinfo (`stdlib/string_dump.go:226,231`); undump discards abslineinfo (`compiler/undump.go:308,315-319`). Negative/wrong error+hook lines after round trip. | `suite-load-dump--dump-lineinfo-int8-wrap.md` |
| 5 | **goto-continue inside a generic for prematurely closes the loop's closing value** (4th explist value): `for line in io.lines(p) do … goto cont … ::cont:: end` dies with "attempt to use a closed file". `compileGotoStmt` placeholder OP_CLOSE patched with a level counting only named locals, landing at/below the for-state registers. Also observable double-`__close`. | `scoping-close--double-close-goto-genfor.md` |

### Tier 2: wrong behavior
| # | Finding | File |
|---|---------|------|
| 6 | `<close>` vars in C-callback frames (gsub/sort) close **eagerly when an error escapes the coroutine** instead of at `coroutine.close` | `coroutines--tbc-eager-close-on-failed-yield.md` |
| 7 | `os.exit()` inside a coroutine is swallowed as a catchable resume error; process keeps running | `coroutines--os-exit-inside-coroutine-swallowed.md` |
| 8 | `table.remove(t, #t+1)` early-returns nil instead of the `lua_geti`/`lua_seti` pair — wrong result when `__len` understates raw contents; skips `__index`/`__newindex` on proxies (`stdlib/table.go` ~212) | `tables-stdlib--remove-pos-len-plus-1-ignores-index.md` |
| 9 | `string.gsub` with a **number subject returns the number** (not a string) when nothing changed | `strings-patterns--gsub-number-subject-returns-number.md` |
| 10 | **Thread values accepted** by table.unpack/sort/concat/move-src and ipairs (mis-messaged by insert/remove/move-dst) — missing type-check family | `coroutines--thread-accepted-by-table-lib.md` |
| 11 | Method-call chains compile inside-out at 2 registers/link: valid chains ≥128 links rejected "too many registers" (reference runs 3000+ in O(1) registers; plain `f()()()` chains fine) | `codegen-reg--method-chain-register-per-link.md` |
| 12 | Named vararg table (`...t`) eagerly materialized into the local slot; ref 5.5 keeps it virtual (PF_VATAB). Observable via debug.getlocal/setlocal. (Memory: known deferral — now pinned to exact divergence.) | `scoping-close--named-vararg-getlocal-lazy.md` |
| 13 | Lexer accepts non-standard `0b101` binary literals (dedicated `scanBinaryNumber`, looks intentional) — **decide: document in wontfix/ or remove** | `numerics--binary-literal-0b-accepted.md` |
| 14 | `debug.getupvalue` on a gmatch iterator returns nil for upvalue 3 where reference returns the state userdata | `strings-patterns--gmatch-upvalue3-nil-vs-userdata.md` |

### Tier 3: CLI surface (official-suite blockers)
| # | Finding | File |
|---|---------|------|
| 15 | CLI `arg` table lacks `arg[-1]`/negative indices (interpreter path) — breaks suite progname discovery | `suite-load-dump--arg-negative-interpreter-missing.md` |
| 16 | CLI ignores `LUA_INIT` / `LUA_INIT_5_5` (code and `@file` forms) while honoring `LUA_PATH` | `suite-load-dump--cli-ignores-lua-init.md` |
| 17 | CLI cannot execute precompiled binary chunk files (`loadfile()` on the same file works) | `suite-load-dump--cli-rejects-binary-chunk-file.md` |

## Error-message-only (note, low priority)
- Unary-op (`#`,`-`,`~`) runtime error misattributes temp operand to destination local on re-assignment (`fresh-perf--len-error-misattributes-dest-local.md`) — in-place-binop codegen artifact.
- Operands flowing through `and`/`or` lose the `(local/upvalue/global 'name')` suffix (found independently by 2 lenses: `codegen-reg--andor-operand-varinfo-name.md`, `suite-load-dump--andor-operand-name-attribution.md`).
- checkinteger family: non-integral numeric-string args report "number expected, got string" vs ref "number has no integer representation" — one shared helper, affects string.*/table.*/math.random/ult/utf8.* (`numerics--string-char-noninteger-string-errmsg.md`, `strings-patterns--float-string-int-arg-message.md`).
- `string.byte` huge range: VM-internal stack wording (`strings-patterns--byte-huge-range-stack-error-wording.md`).
- `string.pack` huge `c` size / packsize size print (`strings-patterns--pack-msgonly-huge-c-size-and-size-print.md`).
- Circular require C-stack error loses require's "error loading module…" wrapper (`suite-load-dump--require-cstack-error-unwrapped.md`).

## Rejected / out-of-scope (kept for the record)
- `coroutines--exit-runs-suspended-tbc.md` — WONTFIX-SCOPE (shutdown semantics, Go-GC design).
- `coroutines--nested-resume-no-cstack-limit.md` — WONTFIX-SCOPE (no C stack; golua has its own limits).
- `coroutines--getlocal-extra-temporary.md` — NOT-A-BUG (impl-defined temporaries).
- `strings-patterns--charset-trailing-escape-consumes-closing-bracket.md` — NOT-A-BUG per verifier.

## Official suite status (this run)
bitwise, bwcoercion, calls, code, constructs, coroutine, cstack, errors, goto, literals, locals,
math, nextvar, pm, strings, tpack, utf8, vararg*, verybig **PASS**. attrib passes with `_port=true`
(and golua matches the suite where /usr/bin/lua5.5.0 itself fails at attrib.lua:272). big passes
inside a coroutine (as all.lua runs it). closure/db pass modulo GC-wait loops (out of scope).
files.lua's genuine failure is bug #15 (arg[-1]). main.lua needs CLI flags (-v/-i/-l/-E/-W/stdin)
— only script-visible pieces filed (#15–17). gc/sort remain knownFail.

## Notable clean areas (heavily probed, full parity)
Fused single-capture closures + upvalue identity/join/dump (the fresh f46da9e surface), no-alloc
metamethod dispatch incl. MMBINK/MMBINI orders + yields in metamethods, k-flag constant stores,
32-byte Value int/float equality/hash edges, math.random xoshiro256** seed-exact vs reference,
float formatting/parsing bit-exact, pattern engine core, pack/unpack encoding semantics, table.sort
hostile comparators, `<close>`/`<const>`/global-decl semantics (except #5), require/searchers,
dump round-trips (except #4), ~150k generated differential programs total.
