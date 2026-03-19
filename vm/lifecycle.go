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

// registerProvider adds a provider to the VM's registered list for
// lifecycle management. Called by each Set*Provider method.
func (vm *VM) registerProvider(p any) {
	vm.registeredProviders = append(vm.registeredProviders, p)
}

// Close shuts down the VM by calling Shutdown on any registered providers
// that implement the Shutdownable interface. Returns the first error encountered.
func (vm *VM) Close(ctx context.Context) error {
	var firstErr error
	for _, p := range vm.registeredProviders {
		if s, ok := p.(Shutdownable); ok {
			if err := s.Shutdown(ctx); err != nil && firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}
