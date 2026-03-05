# Contributing to GoLua

PRs are welcome. This document outlines the contribution workflow and guidelines.

## Pull Request Workflow

1. **Write a failing test** that reproduces the bug or demonstrates the new behavior
2. **Verify the test fails** before implementing the fix
3. **Implement the fix or feature**
4. **Verify against reference Lua 5.4.8** if applicable (`lua5.4` binary)
5. **Run `go test ./... -count=1`** to confirm all tests pass
6. **Submit the PR** with a clear description of what changed and why

## Test Requirements

Bug fixes must include a test that:
- Fails before the fix
- Passes after the fix
- Verifies the behavior matches Lua 5.4.8 where applicable

Preferred test formats:
- **Doctests** (`tests/doctest/*.lua`) — for testing observable `print()` output
- **Stdlib tests** (`tests/stdlib/test_*.lua`) — for regression tests using `assert()`
- **Go tests** (`vm/*_test.go`) — for VM internals that need Go-level control

See [TESTING.md](TESTING.md) for full details on the test infrastructure.

## Coding Guidelines

- **Small commits** with clear, descriptive messages
- **Lua 5.4 semantic compatibility** is the primary goal
- Avoid over-engineering — only implement what is needed
- No unnecessary abstractions or premature optimization
- Follow existing code patterns and conventions
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
- **Additional stdlib functions** — must match Lua 5.4 semantics
- **Test coverage** — especially edge cases from the Lua 5.4 test suite
- **Examples** — small, self-contained, runnable demonstrations
- **Bug fixes** — especially with reference Lua 5.4.8 verification

## Out of Scope

The following are intentionally excluded:
- C shared object loading (`.so`/`.dll`)
- Binary chunk loading (`load(string.dump(f))` round-trip)
- Matching C Lua's garbage collector behavior (Go GC is used)
- Features from Lua versions other than 5.4 (unless also in 5.4)

## Getting Help

If you're unsure about an approach, open an issue to discuss before writing code.
