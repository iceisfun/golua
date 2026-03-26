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
//
// If the provider implements Initializable, Initialize is called with the
// VM's context. If Initialize returns an error, the provider is not
// registered and registerProvider returns the error.
func (vm *VM) registerProvider(p any) error {
	if init, ok := p.(Initializable); ok {
		if err := init.Initialize(vm.ctx); err != nil {
			return err
		}
	}
	vm.registeredProviders = append(vm.registeredProviders, p)
	return nil
}

// Close shuts down the VM by calling Shutdown on any registered providers
// that implement the Shutdownable interface, then runs close hooks and
// discards pending GC finalizers. Returns the first error encountered.
func (vm *VM) Close(ctx context.Context) error {
	var firstErr error
	for _, p := range vm.registeredProviders {
		if s, ok := p.(Shutdownable); ok {
			if err := s.Shutdown(ctx); err != nil && firstErr == nil {
				firstErr = err
			}
		}
	}

	// Run close hooks (only root VM has these; coroutine VMs have nil closeHooks).
	for _, hook := range vm.closeHooks {
		hook(ctx)
	}
	vm.closeHooks = nil

	// Discard pending GC entries to prevent leaks after shutdown.
	if vm.gcQueue != nil {
		vm.gcQueue.mu.Lock()
		vm.gcQueue.pending = nil
		vm.gcQueue.mu.Unlock()
	}

	return firstErr
}
