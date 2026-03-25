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
- `LuaIoProvider` — controls all `io.*` operations, `os.remove`, `os.rename`, `os.tmpname`
- `LuaOsProvider` — controls `os.clock`, `os.time`, `os.date`, `os.getenv`
- `LuaExecProvider` — controls `os.execute`
- `LuaExitHandler` — controls `os.exit`
- `LuaDebugProvider` — gates individual debug library capabilities
- `LuaProcessProvider` — controls `exec.run`, `exec.spawn`, `exec.run_shell` (extension)
- `LuaChanProvider` — Go↔Lua channel communication (extension)
- `LuaTimeProvider` — millisecond timing (extension)
- `LuaPrintProvider` — `print()`/`warn()` output routing
- `LuaLoadLibProvider` — controls `package.loadlib` for host-defined native modules

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

## Binary Chunk Loading

**Behavior:** `string.dump()` produces Lua 5.4 format bytecode, and `load()`, `loadfile()`, and `dofile()` can reload binary chunks. The `mode` parameter (`"b"`, `"t"`, `"bt"`) controls whether binary, text, or both are accepted, matching Lua 5.4 semantics.

**Implementation:** `compiler/undump.go` implements a full Lua 5.4 binary chunk deserializer. Header fields (version, format, sizes, endianness) are validated before loading.

## UTF-8 Extended Range

**Behavior:** The `utf8` library supports both strict and lax modes, matching Lua 5.4:

- **Strict mode** (default) validates standard Unicode (U+0000 to U+10FFFF) using Go's `unicode/utf8` package.
- **Lax mode** accepts the full extended range (up to 0x7FFFFFFF) using custom `appendExtendedUTF8`/`decodeExtendedUTF8` codecs that produce 1–6 byte sequences per RFC 2279.
- `utf8.char` always supports the full extended range regardless of mode.

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
| `exec`    | Process execution with streaming I/O and spawn |
| `http`    | HTTP client (separate module in `stdlib/http`) |
| `bit32`   | Lua 5.2 compatibility library (deprecated in Lua 5.3+) |

These extensions are either absent by default (requiring a provider) or clearly namespaced to avoid conflicts with standard Lua code.
