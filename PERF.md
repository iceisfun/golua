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
