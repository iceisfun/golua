# Contributing to GoLua

PRs are welcome. This document outlines the contribution workflow and guidelines.

## Branches & Versions

GoLua is maintained on two long-lived branches, each tracking a different
PUC-Rio Lua release:

| Branch | Lua compat | Go module | Notes |
|---|---|---|---|
| `master` | Lua 5.5.0 | `github.com/iceisfun/golua/v2` (v2) | Default branch — new work lands here first |
| `lua_5_4_8` | Lua 5.4.8 | `github.com/iceisfun/golua` (v1) | Stable; receives backports |

**Develop on `master` (Lua 5.5).** When a fix is not 5.5-specific, backport it
to `lua_5_4_8`. Some changes are intentionally one-branch-only:
- **5.5-only semantics** (e.g. the new `#`-border behavior, `string.dump` chunk
  reuse, named varargs) stay on `master`.
- **5.4-only semantics** that 5.5 changed stay on `lua_5_4_8`.

When in doubt, verify the expected behavior against the matching reference
binary before deciding where a change belongs.

## Pull Request Workflow

1. **Write a failing test** that reproduces the bug or demonstrates the new behavior
2. **Verify the test fails** before implementing the fix
3. **Implement the fix or feature**
4. **Verify against the matching reference Lua** if applicable (see below)
5. **Run `go test ./... -count=1`** to confirm all tests pass
6. **Run `go vet ./...`** and ensure no warnings
7. **Submit the PR** with a clear description of what changed and why

### Reference Lua binaries

Parity work is differential-tested against the upstream PUC-Rio interpreter.
The relevant binaries:

- **Lua 5.5.0** — `lua5.5.0` on `PATH` (use this for `master`)
- **Lua 5.4.x** — `lua5.4` / `lua` on `PATH` is **5.4.6**, not 5.4.8. For exact
  5.4.8 parity, build from source (`lua-5.4.8`) rather than trusting the `PATH`
  binary.

Always confirm which version you are comparing against — 5.4 and 5.5 diverge in
several observable behaviors (`#` borders, `pairs`/`__pairs` arity, removed math
functions, error message wording, bytecode format).

## Test Requirements

Bug fixes must include a test that:
- Fails before the fix
- Passes after the fix
- Verifies the behavior matches the reference Lua for the branch you are on

Preferred test formats:
- **Doctests** (`tests/doctest/*.lua`) — for testing observable `print()` output
- **Stdlib tests** (`tests/stdlib/*.lua`) — for regression tests using `assert()`
- **Go tests** (`vm/*_test.go`, root-level `*_test.go`) — for VM internals or
  tests that need Go-level control (providers, coroutines, finalizers)

See [TESTING.md](TESTING.md) for full details on the test infrastructure.

> Note: some legacy `tests/doctest/*.lua` files encode pre-5.4 semantics. When
> in doubt, verify expected output against the reference binary, not against an
> existing in-tree test.

## Coding Guidelines

- **Small commits** with clear, descriptive messages
- **Lua semantic compatibility** is the primary goal — 5.5 on `master`, 5.4.8 on
  `lua_5_4_8`
- Avoid over-engineering — only implement what is needed
- No unnecessary abstractions or premature optimization
- Follow existing code patterns and conventions
- Respect the **provider/capability model** — stdlib features are gated behind
  capability interfaces (code, io, os, chan) so embedders can sandbox them
- Run `go vet ./...` and ensure no warnings

## Commit Messages

Use the format: `type(scope): short description`

Examples:
```
fix(vm): correct floor division for negative operands
feat(stdlib): add string.pack and string.unpack
test(stdlib): add regression test for table.sort with __lt
docs(README): update feature list
```

## Areas Accepting Contributions

- **Performance improvements** — benchmarks welcome, profile before optimizing
- **Documentation** — corrections, clarifications, examples
- **Additional stdlib functions** — must match the reference Lua semantics
- **Test coverage** — especially edge cases from the upstream Lua test suite
- **Examples** — small, self-contained, runnable demonstrations
- **Bug fixes** — especially with reference-Lua differential verification

## Out of Scope

The following are intentionally excluded:
- C shared object loading (`.so`/`.dll`) — pure Go, no cgo
- Matching C Lua's garbage collector behavior (Go's GC is used)
- Features from Lua versions other than the branch's target (5.5 on `master`,
  5.4.8 on `lua_5_4_8`), unless present in that target

## Getting Help

If you're unsure about an approach, open an issue to discuss before writing code.
