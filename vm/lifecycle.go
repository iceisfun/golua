package vm

import "context"

// Initializable is an optional interface that providers can implement.
// When a provider implementing Initializable is set on a VM, Initialize
// is called with the VM's context. This allows providers to perform
// startup tasks like opening connections or starting background goroutines.
type Initializable interface {
	Initialize(ctx context.Context) error
}

// Shutdownable is an optional interface that providers can implement.
// When VM.Close is called, Shutdown is called on each provider that
// implements this interface. This allows providers to release resources,
// close connections, or stop background goroutines.
type Shutdownable interface {
	Shutdown(ctx context.Context) error
}

// Close shuts down the VM by calling Shutdown on any providers that
// implement the Shutdownable interface. Returns the first error encountered.
func (vm *VM) Close(ctx context.Context) error {
	providers := []interface{}{
		vm.codeProvider,
		vm.ioProvider,
		vm.osProvider,
		vm.execProvider,
		vm.exitHandler,
		vm.debugProvider,
		vm.chanProvider,
		vm.timeProvider,
		vm.loadLibProvider,
		vm.processProvider,
		vm.printProvider,
	}
	var firstErr error
	for _, p := range providers {
		if p == nil {
			continue
		}
		if s, ok := p.(Shutdownable); ok {
			if err := s.Shutdown(ctx); err != nil && firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}
