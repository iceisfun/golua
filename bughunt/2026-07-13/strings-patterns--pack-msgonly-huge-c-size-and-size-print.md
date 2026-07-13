# string.pack message-only divergences: huge 'c' size, and overflow-truncated size in "out of limits"

**Severity: error-message-only** (both cases raise catchable errors on the same inputs)

## Repro 1 — huge fixed-size 'c' directive

```lua
print(pcall(string.pack, "c1000000000000000", "x"))
```

golua: `false  bad argument #1 to 'string.pack' (result too long)`
lua5.5.0: `false  not enough memory`

golua's deliberate 1<<30 sandbox cap fires first with the reference's
"result too long" wording for the representable-but-too-big case; reference
instead attempts the allocation and reports LUA_ERRMEM. (The cap itself is
the documented sandbox policy — only the message class differs.)

## Repro 2 — integral size printed after C int truncation

```lua
print(pcall(string.packsize, "i99999999999999999999"))
```

golua: `false  integral size (999999999999999999) out of limits [1,16]`
lua5.5.0: `false  integral size (-1486618625) out of limits [1,16]`

Both consume the same 18 digits (same getnum stop rule); reference then
prints the size through a C `int`, truncating 999999999999999999 to
-1486618625. golua prints the true parsed value. golua is arguably more
correct; recorded for fuzzer-noise triage.

## Verification: REJECTED (ERROR-MSG-ONLY)

Verified 2026-07-13 against `/usr/bin/lua5.5.0` (both under `ulimit -v 16GB`,
`timeout 15`). Outputs reproduce exactly as reported. Not in `wontfix/`, not
GC-dependent. Both cases are message-prose-only; no semantic divergence exists
or can exist:

- **Repro 1**: reference passes its own `result too long` check
  (`lstrlib.c:1624`, 1e15 < MAX_SIZE) and dies allocating the pad buffer →
  bare `not enough memory` (environment/ulimit-dependent LUA_ERRMEM). golua's
  1<<30 sandbox cap rejects earlier with an argument error. Both return
  `false, <string>` from pcall. The cap is documented deliberate sandbox
  policy (uncatchable Go runtime OOM avoidance).
- **Repro 2**: in reference `getnumlimit` (`lstrlib.c:1470-1475`) the limit
  check uses the full `size_t sz`; only the *message* passes `sz` through
  `%d` (size_t→int vararg truncation, UB in C). The truncation can never wrap
  a huge size into the accepted [1,16] range because the check happens on the
  untruncated value — verified boundary parity: `i16` accepted, `i17`
  rejected identically on both interpreters. Matching the reference number
  would mean replicating C undefined behavior in an error string.

Minimized repro (already minimal — each line is one pcall):

```lua
print(pcall(string.pack, "c1000000000000000", "x"))
print(pcall(string.packsize, "i99999999999999999999"))
```
