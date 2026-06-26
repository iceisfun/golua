# Table Example

Demonstrates the `LuaTable` interface and deterministic table iteration.

## LuaTable Interface

```go
type LuaTable interface {
    Get(key Value) Value
    // Set assigns val to key (Nil val removes it). Errors on an invalid key (nil/NaN).
    Set(key Value, val Value) error
    // Delete removes key. Errors on an invalid key (nil/NaN).
    Delete(key Value) error
    // Next returns the pair after key (pass Nil for the first). Returns (Nil, Nil, nil) at the end.
    Next(key Value) (nextKey Value, val Value, err error)
    Len() int
    Metatable() LuaTable
    SetMetatable(mt LuaTable)
    // ... coroutine-thread methods (IsThread, VMRef) omitted
}
```

The mutating methods return an `error` so Go callers can react to invalid keys
without a panic crossing the boundary (see `main.go`, which checks every `Set`).

The default implementation (`*Table`) stores hash keys in an ordered slice, so `Next()` yields them in insertion order. This makes iteration deterministic as long as the table is not modified during traversal.

## Iteration Guarantees

- **Deterministic order**: The same table iterated twice (without modification) produces the same key sequence.
- **Insertion-ordered hash part**: Hash keys appear in the order they were first inserted.
- **Array-first traversal**: `Next(nil)` visits the array part (integer keys 1..n) before the hash part.
- **No Go map ordering**: The implementation does not rely on Go's `range map` ordering.

## Important Notes

- **Mutation invalidates iteration**: Inserting or deleting keys during a `Next()` traversal may skip keys or produce duplicates.
- **No implicit thread safety**: Concurrent reads are safe, but concurrent read+write requires external synchronization.
- `Len()` returns the array part length (the `#` operator), not the total number of entries.

## Running

```bash
go run ./examples/table
```
