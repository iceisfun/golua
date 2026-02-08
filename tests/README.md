# Lua Test Suite

This directory contains Lua test files that are automatically discovered and run by Go's test framework.

## Running Tests

```bash
go test ./tests/...        # Run all Lua tests
go test -v ./tests/...     # Verbose output
go test ./...              # Run all tests including other packages
```

## Test File Naming

- `test_*.lua` - Regular tests that should pass
- `broken_*.lua` - Known broken tests (skipped, for tracking issues)

## Writing Tests

Tests are pure Lua files that use `assert()` for validation:

```lua
-- test_example.lua
-- Brief description of what this tests

local result = some_function()
assert(result == expected, "Optional error message")
```

**Guidelines:**
- Use `assert()` for all checks - don't use `print()` for output
- Add a comment at the top describing what the test covers
- Keep tests focused on one feature/behavior
- Use descriptive file names: `test_feature_name.lua`

## Broken Tests

When a test fails due to a known bug or unimplemented feature, prefix it with `broken_`:

```lua
-- broken_feature_x.lua
-- BROKEN: Description of what's broken
-- Issue: Link or description of the issue
-- Expected: What should happen
-- Actual: What currently happens

-- The failing test code
assert(unimplemented_feature() == 42)
```

Broken tests are automatically skipped but tracked. When the issue is fixed, rename the file to `test_*.lua`.

## Benchmarking

```bash
go test -bench=. ./tests/...
```
