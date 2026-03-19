# Design Exceptions and Rationale

This document describes intentional differences between GoLua and the reference C Lua 5.4 implementation, with rationale for each decision.

## Garbage Collector

**Difference:** GoLua delegates garbage collection entirely to Go's runtime GC. The `collectgarbage()` function is implemented but does not provide deterministic step/generational behavior.

**What works:**
- `collectgarbage("collect")` triggers `runtime.GC()`
- `collectgarbage("count")` returns approximate memory usage via `runtime.MemStats`
- `collectgarbage("stop")`, `collectgarbage("restart")`, `collectgarbage("incremental")`, `collectgarbage("generational")` are accepted without error
- Weak tables (`__mode = "k"`, `"v"`, `"kv"`) are supported via Go finalizers

**What differs:**
- Exact finalization timing is non-deterministic
- `collectgarbage("step")` does not provide fine-grained control
- Tests that depend on specific GC ordering or finalization counts may not pass
- `__gc` metamethods on tables are supported but timing is Go-GC-dependent

**Rationale:** Go manages memory through its own garbage collector. Implementing a separate allocator and GC would add enormous complexity for minimal practical benefit. Correctness and eventual finalization are guaranteed.

## Provider Architecture

**Difference:** GoLua uses provider interfaces rather than direct OS/filesystem access. The VM starts sandboxed with no host access.

**C Lua behavior:** Links directly against libc for file I/O, OS functions, and `loadlib`.

**GoLua behavior:** Each capability is opt-in:
- `LuaCodeProvider` — controls `dofile`, `loadfile`, `require` file searching
- `LuaIoProvider` — controls all `io.*` operations
- `LuaOsProvider` — controls `os.*` operations
- `LuaDebugProvider` — gates individual debug library capabilities
- `LuaChanProvider` — Go↔Lua channel communication (extension)
- `LuaTimeProvider` — millisecond timing (extension)
- `LuaPrintProvider` — `print()`/`warn()` output routing

**Rationale:** Embedding Lua in Go applications requires control over side effects. The provider model enables sandboxing, testability, and clean separation between the interpreter and the host environment.

## Context Threading

**Design:** All provider interface methods receive a `context.Context` as their first parameter. The VM stores a context internally (defaulting to `context.Background()` when created with `New()`), and stdlib functions pass `v.Context()` through to provider calls.

**Flow:** `context.Context` → `VM.SetContext()` → Lua call → native stdlib function → `provider.Method(ctx, ...)`

This allows Go code embedding GoLua to propagate cancellation and deadlines through the entire call chain. A caller can set a context with a timeout on the VM, and any provider method (file I/O, OS operations, process execution) will observe that cancellation.

**Lifecycle hooks:** Providers may optionally implement `Initializable` (setup) and `Shutdownable` (teardown) interfaces via type assertion, enabling resource management tied to the VM lifecycle.

**Rationale:** Go's `context.Context` is the standard mechanism for cancellation and deadline propagation. Threading it through providers gives embedders control over long-running or blocking operations without requiring Lua code awareness.

## No C Module Loading

**Difference:** `require` does not load C shared objects (`.so`/`.dll`). The C file searcher always returns "not found".

**Rationale:** GoLua is pure Go with no cgo dependency. Loading C shared libraries would require cgo and would undermine the sandbox model. Lua modules (`.lua` files) are fully supported via `LuaCodeProvider`.

## No Binary Chunk Loading

**Difference:** `string.dump()` produces bytecode output, but `load()` cannot reload binary chunks. Only source strings are accepted by `load()`.

**Rationale:** Binary chunk loading requires implementing a bytecode deserializer and verifier, which is a large attack surface for minimal benefit. Source-level loading is sufficient for all standard use cases.

## UTF-8 Strict Mode

**Difference:** The `utf8` library enforces strict Unicode validation (U+0000 to U+10FFFF). The `lax` parameter is accepted for API compatibility but still uses Go's strict validation.

**What differs:**
- `utf8.char` rejects codepoints above U+10FFFF (Lua accepts up to 0x7FFFFFFF)
- Surrogates (U+D800-U+DFFF) are always rejected
- `lax` mode may still error on invalid sequences

**Rationale:** Go's `unicode/utf8` package enforces RFC 3629 (4-byte max, U+10FFFF limit). Implementing a custom UTF-8 encoder for non-standard codepoints would violate the project constraint of using only Go's standard library. The practical impact is negligible — codepoints above U+10FFFF are not valid Unicode.

See [docs/utf8.md](utf8.md) for the full analysis.

## Table Iteration Order

**Difference:** `next()`/`pairs()` iteration over tables is deterministic (insertion-ordered). C Lua uses hash table order, which is deterministic but not insertion-ordered.

**Rationale:** Go's `map` type has randomized iteration. GoLua uses an ordered keys slice to ensure reproducible behavior across runs. This is stricter than Lua's guarantee (Lua only promises that all keys are visited) but never breaks correct Lua programs.

## Debug Hook Instruction Counts

**Difference:** Count hook instruction counts may differ from reference Lua due to extra `CLOSE` instructions emitted by the compiler for `for` loops.

**Rationale:** The GoLua compiler emits `CLOSE` instructions in some cases where C Lua's compiler optimizes them away. This affects only instruction counting in debug hooks, not program semantics.

## Coroutine Thread Type

**Difference:** Coroutine threads are represented as tables with an `IsThread()` flag, rather than as a distinct type. `type()` returns `"thread"` for coroutine objects.

**Rationale:** The `LuaTable` interface is GoLua's universal container. Representing threads as tables with a flag avoids adding a separate type to the value system while maintaining correct `type()` behavior.

## math.random Isolation

**Difference:** Each VM has its own independent random state. `math.randomseed()` in one VM does not affect others.

**C Lua behavior:** Uses global C `rand()`/`srand()` state (Lua 5.4 uses its own generator, but the state is per-lua_State).

**Rationale:** Per-VM isolation prevents cross-contamination in multi-VM environments, which is the common embedding pattern in Go applications.

## Non-Standard Extensions

GoLua includes several extensions not present in standard Lua:

| Extension | Description |
| --------- | ----------- |
| `glob`    | Go-style case-insensitive pattern matching |
| `chan`     | Go↔Lua channel-based message passing |
| `time`    | Millisecond-precision timing and periodic triggers |
| `bit32`   | Lua 5.2 compatibility library (deprecated in Lua 5.3+) |

These extensions are either absent by default (requiring a provider) or clearly namespaced to avoid conflicts with standard Lua code.
