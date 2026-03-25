# Lifecycle Interfaces

Providers can optionally implement `Initializable` and `Shutdownable` for setup and teardown. These are defined in `vm/lifecycle.go`.

## Interfaces

```go
// Initializable is called when a provider is set on a VM.
type Initializable interface {
    Initialize(ctx context.Context) error
}

// Shutdownable is called when VM.Close is invoked.
type Shutdownable interface {
    Shutdown(ctx context.Context) error
}
```

## Initializable

When a provider implementing `Initializable` is set on a VM (e.g. via `v.SetIoProvider`), its `Initialize` method is called with the VM's context. Use this for startup tasks like opening connections or verifying configuration.

If `Initialize` returns an error, the setter method returns that error, the provider field is cleared, and the provider is not registered for shutdown.

## Shutdownable

When `v.Close(ctx)` is called, the VM iterates all registered providers and calls `Shutdown` on each one that implements `Shutdownable`. This allows providers to release resources, close connections, or stop background goroutines.

`Close` returns the first error encountered but continues shutting down remaining providers.

## Example

```go
type DBProvider struct {
    db *sql.DB
}

func (p *DBProvider) Initialize(ctx context.Context) error {
    return p.db.PingContext(ctx)
}

func (p *DBProvider) Shutdown(ctx context.Context) error {
    return p.db.Close()
}
```

## Usage

Always call `v.Close(ctx)` when done with a VM:

```go
v := vm.New(vm.WithContext(ctx))
if err := v.SetIoProvider(vm.NewFullIoProvider("/app/data")); err != nil {
    log.Fatalf("provider init failed: %v", err)
}
stdlib.Open(v)

_, err := v.Run(proto)
// ... handle err ...

if err := v.Close(ctx); err != nil {
    log.Printf("cleanup error: %v", err)
}
```

Providers that do not implement either interface are silently skipped.
