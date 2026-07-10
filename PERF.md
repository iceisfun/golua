# GoLua Performance Benchmarks

## Running Benchmarks

```bash
go test -bench=. -benchmem -count=6 ./bench/ | tee bench.txt
```

The `-count=6` flag runs each benchmark 6 times for statistical significance.

## Comparing Results

Use [benchstat](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat) to compare runs:

```bash
# Save baseline
git stash  # or checkout baseline commit
go test -bench=. -benchmem -count=6 ./bench/ > old.txt

# Switch to new code
git stash pop  # or checkout new commit
go test -bench=. -benchmem -count=6 ./bench/ > new.txt

# Compare
benchstat old.txt new.txt
```

Install benchstat:

```bash
go install golang.org/x/perf/cmd/benchstat@latest
```

## Benchmark Suite

| Benchmark | Script | Subsystems |
|---|---|---|
| Primes | primes.lua | numeric for, integer arithmetic, branch, table |
| Sort | sort.lua | table indexing, function calls, comparator |
| BinaryTrees | binarytrees.lua | recursion, allocation, GC |
| NBody | nbody.lua | floating-point math, table reads, register pressure |
| TableChurn | table_churn.lua | table alloc/insert, GC |
| CoroutinePingPong | coroutine_pingpong.lua | coroutine create/resume/yield |
| StringConcat | string_concat.lua | string alloc, concat operator, GC |
| ClosureAlloc | closure_alloc.lua | closure creation, upvalue capture |
| Metamethod | metamethod.lua | __add dispatch, dynamic call overhead |

## Metrics

Each benchmark reports three metrics via `go test -benchmem`:

- **ns/op** - wall-clock time per iteration
- **B/op** - bytes allocated per iteration
- **allocs/op** - heap allocations per iteration

## Methodology

- Each benchmark compiles the Lua script once, then runs it N times on a fresh VM.
- The stdlib is loaded for every iteration to reflect realistic usage.
- `b.ResetTimer()` excludes compilation from timing.
- Scripts use fixed seeds or deterministic inputs for reproducibility.

## Reading sec/op: NBody and BinaryTrees are bimodal on code placement

Some benchmarks in this suite report a sec/op that depends on *where* code lands
in the binary, not only on what the code does. NBody has two stable modes about
10% apart (roughly 47.9 ms/op and 43.8 ms/op); BinaryTrees moves a few percent.
A change that touches an early file in `vm/` can shift the interpreter's hot
functions between modes without altering a single executed instruction.

Pinning (`taskset` + a fixed `GOMAXPROCS`) removes thermal noise but does *not*
remove this. `benchstat` will happily report `p=0.000` on the artifact.

Before crediting a change with a sec/op win on these two:

1. Check whether the machine code even changed:
   `go tool nm -size ./bench.test | grep -E 'vm\.\(\*VM\)\.(execute|call|doCall|arith)$'`
   Identical sizes across the two builds means identical code, so any delta is
   placement.
2. Run a **pad sweep** control. Insert an inert, never-executed function into an
   early file of the `vm` package (`vm/function.go` works; files that sort after
   `vm_exec.go`, such as `vm_upvalue.go`, shift nothing) and rebuild at several
   pad sizes. If the *unmodified* baseline reaches the "optimized" time, the win
   is placement.

The pad must survive dead-code elimination, or the binary is unchanged and the
control silently passes. An unexported, unreferenced function is deleted by the
linker. Guard the call with a package-level `var` that is never set:

```go
var padEnabled bool // never set; keeps padNeverCalled reachable
var padSink int

//go:noinline
func padNeverCalled(n int) int {
    s := 0
    for i := 0; i < n; i++ {
        s ^= s >> 2
        s = s*31 + i
    }
    return s
}

// at the top of NewClosure:
if padEnabled {
    padSink = padNeverCalled(len(proto.Upvalues))
}
```

Confirm it is actually in the binary: `go tool nm bench.test | grep padNeverCalled`
must print one symbol, not zero.

What to trust instead:

- `allocs/op` and `B/op` are deterministic and reproduce exactly. Prefer them.
- A sec/op delta is credible when it tracks a real allocation change *and*
  reproduces on both the `master` (5.5) and `lua_5_4_8` branches, whose diverged
  VM cores give different code layouts.

## Benchmark Helpers

The `bench` table is available in benchmark Lua scripts (injected by the Go harness):

- `bench.gc()` - force a Go GC cycle
- `bench.consume(value)` - prevent dead-code elimination

These are not part of the standard library and only exist in the benchmark harness.

## Tips

- Close other workloads when benchmarking for stable results.
- Use `-benchtime=3s` for longer runs if results are noisy.
- Use `-run=^$` to skip unit tests: `go test -run=^$ -bench=. -benchmem ./bench/`
- Pin to a single CPU for less variance: `GOMAXPROCS=1 go test -bench=. ./bench/`
