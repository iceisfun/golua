package vm

import (
	"context"
	"errors"
	"testing"
)

// trackingProvider records Initialize and Shutdown calls.
type trackingProvider struct {
	initialized bool
	shutdown    bool
	initCtx     context.Context
	shutCtx     context.Context
	initErr     error
	shutErr     error
}

func (p *trackingProvider) Initialize(ctx context.Context) error {
	p.initialized = true
	p.initCtx = ctx
	return p.initErr
}

func (p *trackingProvider) Shutdown(ctx context.Context) error {
	p.shutdown = true
	p.shutCtx = ctx
	return p.shutErr
}

// Implement LuaDebugProvider so we can use SetDebugProvider.
func (p *trackingProvider) Capabilities(ctx context.Context) LuaDebugCaps {
	return LuaDebugCaps{}
}

func TestInitializableCalledOnSet(t *testing.T) {
	v := New()
	p := &trackingProvider{}
	v.SetDebugProvider(p)

	if !p.initialized {
		t.Fatal("Initialize was not called when provider was set")
	}
}

func TestInitializableReceivesVMContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxKey("test"), "val")
	v := New(WithContext(ctx))
	p := &trackingProvider{}
	v.SetDebugProvider(p)

	if p.initCtx != ctx {
		t.Fatal("Initialize did not receive the VM's context")
	}
}

type ctxKey string

func TestInitializableErrorPreventsRegistration(t *testing.T) {
	v := New()
	p := &trackingProvider{initErr: errors.New("init failed")}
	v.SetDebugProvider(p)

	// Provider should not be registered for shutdown
	if err := v.Close(context.Background()); err != nil {
		t.Fatalf("Close should return nil when provider was not registered: %v", err)
	}
	if p.shutdown {
		t.Fatal("Shutdown should not be called on a provider that failed Initialize")
	}

	// Provider should not be set on the VM
	if v.DebugProvider() != nil {
		t.Fatal("provider should not be set when Initialize fails")
	}
}

func TestShutdownableCalledOnClose(t *testing.T) {
	v := New()
	p := &trackingProvider{}
	v.SetDebugProvider(p)

	ctx := context.Background()
	err := v.Close(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !p.shutdown {
		t.Fatal("Shutdown was not called on Close")
	}
	if p.shutCtx != ctx {
		t.Fatal("Shutdown did not receive the Close context")
	}
}

func TestShutdownableReturnsFirstError(t *testing.T) {
	v := New()
	p1 := &trackingProvider{shutErr: errors.New("err1")}
	p2 := &trackingProvider{shutErr: errors.New("err2")}
	v.SetDebugProvider(p1)
	v.SetTimeProvider(p2)

	err := v.Close(context.Background())
	if err == nil {
		t.Fatal("expected error from Close")
	}
	if err.Error() != "err1" {
		t.Fatalf("expected first error, got: %v", err)
	}
	// Both should still be shut down
	if !p1.shutdown || !p2.shutdown {
		t.Fatal("all providers should be shut down even after errors")
	}
}

// Implement LuaTimeProvider so p2 can be used with SetTimeProvider.
func (p *trackingProvider) Now(ctx context.Context) int64    { return 0 }
func (p *trackingProvider) Tick(ctx context.Context, key string, ms int64) bool { return false }
func (p *trackingProvider) Once(ctx context.Context, key string) bool           { return false }

func TestShutdownableWithCancelledContext(t *testing.T) {
	v := New()
	p := &trackingProvider{}
	v.SetDebugProvider(p)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before Close

	err := v.Close(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !p.shutdown {
		t.Fatal("Shutdown should still be called with cancelled context")
	}
	if p.shutCtx.Err() != context.Canceled {
		t.Fatal("Shutdown should receive the cancelled context")
	}
}

func TestNonLifecycleProviderSkipped(t *testing.T) {
	// A provider that implements neither Initializable nor Shutdownable
	// should be silently skipped.
	v := New()
	v.SetDebugProvider(NewDefaultDebugProvider())

	err := v.Close(context.Background())
	if err != nil {
		t.Fatalf("Close should succeed for non-Shutdownable providers: %v", err)
	}
}

func TestCloseWithNoProviders(t *testing.T) {
	v := New()
	err := v.Close(context.Background())
	if err != nil {
		t.Fatalf("Close with no providers should return nil: %v", err)
	}
}
