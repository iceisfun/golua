# GoLua Test Guide

## Test Categories

### 1. Integration Tests (`tests/test_*.lua`)

Full Lua programs that use `assert()` to validate behavior. Run with the full stdlib. Files prefixed `broken_` are skipped as known failures.

```bash
go test ./tests -run TestLuaFiles -v
```

### 2. Broken Tests (`tests/broken_*.lua`, `tests/broken/`)

Known failures or missing features. Automatically skipped. Tracked as issues.

Out of scope items that may or may not be applicable to future work.

### 3. Stdlib Tests (`tests/stdlib/*.lua`)

Regression tests for standard library functions. Each file focuses on one module or function group.

```bash
go test ./tests -run TestStdlib -v
```

### 4. Stress Tests (`tests/stress/*.lua`)

Workload and performance stability tests for tables, loops, and allocations.

```bash
go test ./tests -run TestStress -v
```

### 5. Doctests (`tests/doctest/*.lua`)

Output-driven tests using `-->` directives to specify expected `print()` output:

```lua
print(1 + 2)
--> =3

print(type({}))
--> =table

print(tostring(1/0))
--> ~inf
```

Directive syntax:
- `-->` or `--> =text` — exact match
- `--> ~pattern` — regex match

The doctest harness provides:
- **Timeout**: 10s default, context-based cancellation
- **VM Limits**: MaxCallDepth=200, MaxStackSlots=10000, MaxInstructions=10M, MaxMetaDepth=200
- **Panic Recovery**: classifies results as success, Lua error, VM panic (bug), or timeout
- **Helper functions**: `doctest.*` table (see below)

```bash
go test ./tests -run TestDoctest -v
```

### 6. VM Unit Tests (`vm/*_test.go`)

Go-level tests for VM internals (opcodes, metamethods, limits). Cannot import `stdlib` due to circular dependency — use inline native functions instead.

```bash
go test ./vm -v
```

### 7. Edge Tests (`tests/edge_test.go`)

Go-level integration tests that exercise specific edge cases.

### 8. Proposed Tests (`proposed_tests/*.lua`)

Staging area for new tests before promotion to `tests/doctest/`. Run with the doctest harness.

```bash
go test ./tests -run TestProposed -v
```

## Doctest Helper Functions

Available as the `doctest` global table in all doctest and proposed test files:

| Function | Description |
|----------|-------------|
| `doctest.assert(cond, msg?)` | Fail if `cond` is falsy |
| `doctest.fail(msg)` | Unconditional failure |
| `doctest.expect_error(fn)` | Call `fn()`; fail if no error; return error value |
| `doctest.expect_equal(a, b)` | Fail if `a ~= b` (uses Lua equality) |
| `doctest.expect_type(val, name)` | Fail if `type(val) ~= name` |
| `doctest.set_timeout(seconds)` | Lower the execution timeout (cannot raise) |

## Running All Tests

```bash
go test ./... -count=1
```

## Running Individual Lua Scripts

The CLI supports a `--test` flag for running individual Lua files with testing infrastructure:

```bash
go run ./cmd/lua --test script.lua
```

`--test` enables:
- `DefaultDebugProvider` (full debug library)
- `JailedIoProvider` rooted at the script's directory
- `DirCodeProvider` rooted at the working directory (enables `require` and `dofile`)

## Conformance Status

All imported Lua 5.4 conformance tests pass. The `new/` directory was previously used as a staging area for conformance tests under development — it is now empty, as all tests have been resolved and promoted to `tests/stdlib/`.

## Adding New Tests

Preferred approaches for new tests:

1. **Doctests** (`tests/doctest/*.lua`) — best for testing observable behavior via `print()` output. Use `-->` directives for expected output.
2. **Stdlib tests** (`tests/stdlib/test_*.lua`) — best for regression tests using `assert()`. One file per module or feature.
3. **Go tests** (`vm/*_test.go`, `tests/edge_test.go`) — best for testing VM internals or edge cases that need Go-level control.

For bug fixes, the recommended workflow is:
1. Write a failing test that reproduces the bug
2. Verify the test fails
3. Fix the bug
4. Verify against reference Lua 5.4.8 if applicable
