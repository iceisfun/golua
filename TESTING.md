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

### 9. Embedding, concurrency, and soak tests (root `*_test.go`, `package golua_test`)

Property/stress tests for the Go-specific surfaces that have no PUC-Lua
equivalent (so they use invariants, not differential comparison):

- `embed_api_test.go` — Go↔Lua value-marshaling round-trips, native-function
  calling conventions, calling Lua from Go, and the API-level sandbox guarantee
  (a panicking Go native is a catchable Lua error).
- `concurrency_race_test.go` — many VMs, a shared read-only proto, and channels
  run concurrently. **Run under the race detector** for full value:
  ```bash
  go test . -run 'TestConcurrent|TestEmbed' -race
  ```
- `coroutine_lifecycle_test.go` — pins that completed/closed coroutines reap
  their goroutine (and records the abandoned-suspended leak; see
  `wontfix/coroutine-goroutine-leak`).
- `soak_test.go` — endurance tests, **gated** so they don't run by default. Each
  runs a workload hard for a duration (default 10m) checking a stability
  invariant (determinism / heap-bound / goroutine-bound):
  ```bash
  GOLUA_SOAK=1 GOLUA_SOAK_DURATION=2m go test -run TestSoak -timeout 0 .
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

Without `--test`, the CLI uses its normal interactive environment instead:
- `FullIoProvider` rooted at the script's directory
- `DefaultExecProvider`
- `DefaultExitHandler`
- `DefaultDebugProvider`
- `DirCodeProvider` rooted at the working directory

## Conformance Status

All imported Lua conformance tests pass. Beyond the in-tree suite, behavior is
validated byte-for-byte against the reference interpreters (`lua5.5.0` on
`master`, `lua5.4.8` on `lua_5_4_8`) by the differential / property-based
finders in the sibling [`golua-conformance`](https://github.com/iceisfun/golua-conformance)
repo (pack / pattern / format / coercion / math / date / utf8 / coroutine /
pairs / compiler-limit / debug grinders, plus oracle-free sandbox and bytecode
robustness fuzzers and a semantic program generator). Each divergence those
tools find is pinned here as a doctest or Go test; the small set of *intentional*
divergences is documented in [`wontfix/`](wontfix/).

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
