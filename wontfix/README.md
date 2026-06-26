# wontfix — documented, intentional divergences from reference Lua

This directory records behaviors where **golua deliberately differs from the
PUC-Rio reference interpreter** (`lua5.5.0` / `lua5.4.8`) and we do **not** plan
to change. Each is a real, reproducible difference — but changing it would mean
being *less* correct (IEEE 754), depending on C-library/platform quirks Go does
not expose, or breaking golua's sandbox/embedding design.

These are distinct from bugs. golua's conformance is validated continuously by
the differential fuzzers in
[`golua-conformance`](https://github.com/iceisfun/golua-conformance); the
divergences here are the *known residue* those fuzzers surface, triaged and
explained once so they don't get re-reported.

## Layout

Each subdirectory is one issue:

```
wontfix/<issue-name>/
    example.lua   # runnable; prints golua's output, with the reference output in comments
    README.md     # what it is, why it won't change, and where the behavior lives in the source
```

Run any example against golua and the reference to see the difference:

```sh
go run ./cmd/lua wontfix/<issue-name>/example.lua    # golua
lua5.5.0            wontfix/<issue-name>/example.lua    # reference
```

## Index

| Issue | One-line summary |
|-------|------------------|
| [`signed-zero-const-fold`](signed-zero-const-fold/) | Compile-time `-0.0 - 0` keeps the IEEE sign (`-0.0`); reference's folder yields `+0.0`. |
| [`libm-last-ulp`](libm-last-ulp/) | Transcendental math (`^`, `sin`, `exp`, …) can differ in the last ULP: Go `math` vs C `libm`. |
| [`fmod-nan-sign`](fmod-nan-sign/) | `math.fmod`/`%` with a negative-NaN operand may print `-nan` vs `nan` (C-libm-dependent NaN sign). |
| [`length-operator-border`](length-operator-border/) | `#t` on a table with an interior hole may return a different — but equally valid — *border*. |
| [`os-time-isdst-no-dst-zone`](os-time-isdst-no-dst-zone/) | `os.time{...isdst=true}` under a no-DST zone errors instead of applying glibc's −3600 hack. |
| [`weak-tables-and-gc`](weak-tables-and-gc/) | Weak tables, `__gc`, and collection *timing* follow the Go GC, not Lua's incremental collector. |
| [`untrusted-binary-chunks`](untrusted-binary-chunks/) | Executing a crafted `load(…,"b")` binary chunk can hang/crash — inherently unsafe in golua *and* reference Lua (no bytecode verifier); loading is hardened to a catchable error. |
| [`load-stack-overflow-traceback`](load-stack-overflow-traceback/) | Reference's `load()` embeds a stack traceback into the `"C stack overflow"` error message (golua returns the clean message); plus related compiler-limit near-token and C-stack-vs-fixed-limit wording divergences. |
| [`coroutine-goroutine-leak`](coroutine-goroutine-leak/) | An *abandoned suspended* coroutine (never completed, never `coroutine.close`'d) leaks its backing goroutine — Go can't reap a parked goroutine. Completed/closed coroutines reap fully; reference collects abandoned ones. Mitigation: close or complete coroutines. |

## Filing an issue

If you hit one of the behaviors above, it is **expected** — please read the
corresponding `README.md` rather than opening a report. If you believe a case is
genuinely wrong (not one of these), include a minimal `.lua` and the output of
both golua and `lua5.5.0`; the fastest path to a fix is a failing case in
`golua-conformance`.
