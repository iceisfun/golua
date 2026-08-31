package stdlib

import (
	"reflect"
	"sync"
	"time"

	"github.com/iceisfun/golua/v2/vm"
)

// chanState holds the per-VM chan-module state: the metatable shared by every
// channel handle the VM creates. It lives in the VM's internal state bag, which
// a root VM shares with its coroutine VMs but with no other VM, mirroring
// ioState and matching reference Lua, where a library's handle metatable lives
// in that lua_State's registry.
//
// The metatable must not be a package-level value: a Go process routinely hosts
// many independent VMs, and one process-wide table would let a script running in
// one VM change how handles behave in every other, and let two VMs mutate the
// same *vm.Table concurrently.
type chanState struct {
	meta *vm.Table
}

const chanStateKey = "chan"

// chanStateMu serializes first-use creation of a VM's chanState so that two
// coroutine VMs sharing one internal state bag cannot each install a state (and
// therefore a different handle metatable). It guards creation only; it holds no
// state of its own.
var chanStateMu sync.Mutex

// getChanState returns the per-VM chan state, creating it on first call.
func getChanState(v *vm.VM) *chanState {
	if s := v.InternalState(chanStateKey); s != nil {
		return s.(*chanState)
	}
	chanStateMu.Lock()
	defer chanStateMu.Unlock()
	if s := v.InternalState(chanStateKey); s != nil {
		return s.(*chanState)
	}
	meta := vm.NewEmptyTable()
	meta.SetString(vm.MetaName, vm.NewString("channel"))
	s := &chanState{meta: meta}
	v.SetInternalState(chanStateKey, s)
	return s
}

// openChan registers the chan library if a ChanProvider is set.
func openChan(v *vm.VM) {
	provider := v.ChanProvider()
	if provider == nil {
		return
	}

	caps := provider.Capabilities(v.Context())
	chanTable := vm.NewEmptyTable()

	chanTable.SetString("make", vm.NewNativeFunc(makeChanMake(v, provider)))

	if caps.AllowSelect {
		chanTable.SetString("select", vm.NewNativeFunc(makeChanSelect(v, provider)))
	}

	v.SetGlobal("chan", vm.NewTable(chanTable))
}

// WrapChannel creates a Lua table handle from a Go-created LuaChannel.
func WrapChannel(v *vm.VM, ch *vm.LuaChannel) vm.Value {
	return makeChannelHandle(v, ch)
}

// ProvideChan is a convenience function: sets a DefaultChanProvider and opens the module.
func ProvideChan(v *vm.VM) {
	v.SetChanProvider(vm.NewDefaultChanProvider())
	openChan(v)
}

// Fields a channel handle table carries. chanFieldID is the channel's numeric
// identity, reported for diagnostics. chanFieldChannel holds the channel itself
// as a userdata: it is what chan.select resolves a handle through, so the
// channel a handle names travels with the handle instead of living in a
// side table keyed by the numeric id. That keeps a channel reachable exactly as
// long as its handle is, and means a handle can only name a channel that was
// actually given to the script.
const (
	chanFieldID      = "__chan_id"
	chanFieldChannel = "__chan"
)

// makeChannelHandle creates a Lua table handle wrapping a LuaChannel.
func makeChannelHandle(v *vm.VM, ch *vm.LuaChannel) vm.Value {
	handle := vm.NewEmptyTable()
	handle.SetMetatable(getChanState(v).meta)
	handle.SetString(chanFieldID, vm.NewInt(ch.ID()))
	handle.SetString(chanFieldChannel, vm.NewUserdataValueUV(ch, nil, 0))

	provider := v.ChanProvider()
	caps := provider.Capabilities(v.Context())

	if caps.AllowSend {
		handle.SetString("send", vm.NewNativeFunc(makeChanSend(v, ch)))
	}
	if caps.AllowRecv {
		handle.SetString("recv", vm.NewNativeFunc(makeChanRecv(v, ch)))
	}
	if caps.AllowClose {
		handle.SetString("close", vm.NewNativeFunc(makeChanClose(ch)))
	}
	if caps.AllowTrySend {
		handle.SetString("try_send", vm.NewNativeFunc(makeChanTrySend(ch)))
	}
	if caps.AllowTryRecv {
		handle.SetString("try_recv", vm.NewNativeFunc(makeChanTryRecv(ch)))
	}

	return vm.NewTable(handle)
}

// chanKey is the cached chanFieldChannel key Value. extractChannel runs on
// every chan.select argument, so building the key with vm.NewString per call
// would box the string into Value.ptr (an interface) and heap-allocate each
// time.
var chanKey = vm.NewString(chanFieldChannel)

// extractChannel returns the *LuaChannel a channel handle table carries, or nil
// if the value is not a handle this VM produced.
func extractChannel(handle vm.Value) *vm.LuaChannel {
	t := handle.AsTable()
	if t == nil {
		return nil
	}
	ud := t.Get(chanKey).AsUserdata()
	if ud == nil {
		return nil
	}
	ch, _ := ud.Data.(*vm.LuaChannel)
	return ch
}

// chan.make(size?)
func makeChanMake(v *vm.VM, provider vm.LuaChanProvider) vm.NativeFunc {
	return func(v *vm.VM) int {
		size := 0
		if !v.Get(1).IsNil() {
			s, ok := v.Get(1).ToInt()
			if !ok {
				panic("bad argument #1 to 'chan.make' (number expected)")
			}
			if s < 0 {
				panic("bad argument #1 to 'chan.make' (non-negative size expected)")
			}
			size = int(s)
		}
		ch := provider.NewChannel(v.Context(), size)
		v.Set(0, makeChannelHandle(v, ch))
		return 1
	}
}

// ch:send(val) - blocking, panics on interrupt/closed
func makeChanSend(luaVM *vm.VM, ch *vm.LuaChannel) vm.NativeFunc {
	return func(v *vm.VM) int {
		// v.Get(1) is self (the channel table), v.Get(2) is the value
		val := v.Get(2)
		if err := ch.Send(v.Context(), val); err != nil {
			panic("interrupted")
		}
		if err := v.CheckInterrupt(); err != nil {
			panic("interrupted")
		}
		return 0
	}
}

// ch:recv() -> val, ok - blocking, panics on interrupt
func makeChanRecv(luaVM *vm.VM, ch *vm.LuaChannel) vm.NativeFunc {
	return func(v *vm.VM) int {
		val, ok, err := ch.Recv(v.Context())
		if err != nil {
			panic("interrupted")
		}
		if err := v.CheckInterrupt(); err != nil {
			panic("interrupted")
		}
		v.Set(0, val)
		v.Set(1, vm.NewBool(ok))
		return 2
	}
}

// ch:close() - raises Lua error if already closed
func makeChanClose(ch *vm.LuaChannel) vm.NativeFunc {
	return func(v *vm.VM) int {
		if err := ch.Close(); err != nil {
			panic(err.Error())
		}
		return 0
	}
}

// ch:try_send(val) -> bool
func makeChanTrySend(ch *vm.LuaChannel) vm.NativeFunc {
	return func(v *vm.VM) int {
		val := v.Get(2) // self at 1, val at 2
		ok := ch.TrySend(val)
		v.Set(0, vm.NewBool(ok))
		return 1
	}
}

// ch:try_recv() -> val, ok, received
func makeChanTryRecv(ch *vm.LuaChannel) vm.NativeFunc {
	return func(v *vm.VM) int {
		val, ok, received := ch.TryRecv()
		v.Set(0, val)
		v.Set(1, vm.NewBool(ok))
		v.Set(2, vm.NewBool(received))
		return 3
	}
}

// chan.select(ch1, ch2, ..., timeout?) -> idx, val, ok
// Uses reflect.Select with N recv cases + optional timeout.
// Returns: idx=0 on timeout, idx=1..N for channel match, panics on context cancellation.
func makeChanSelect(luaVM *vm.VM, provider vm.LuaChanProvider) vm.NativeFunc {
	return func(v *vm.VM) int {
		argc := v.ArgCount()
		if argc == 0 {
			panic("bad argument to 'chan.select' (at least one argument expected)")
		}

		// Determine if last arg is a timeout number
		var timeout time.Duration
		hasTimeout := false
		channelCount := argc

		lastArg := v.Get(argc)
		if lastArg.IsNumber() {
			// Last arg is timeout in seconds
			t, _ := lastArg.ToNumber()
			if t < 0 {
				panic("bad argument to 'chan.select' (non-negative timeout expected)")
			}
			timeout = time.Duration(t * float64(time.Second))
			hasTimeout = true
			channelCount = argc - 1
		}

		if channelCount == 0 && !hasTimeout {
			panic("bad argument to 'chan.select' (at least one channel expected)")
		}

		// Build reflect.Select cases
		// Layout: [ctx.Done (if ctx set)] [ch1, ch2, ...] [timeout (if set)]
		cases := make([]reflect.SelectCase, 0, channelCount+2)
		channelOffset := 0

		// Context cancellation case — always present since v.Context() is never nil
		ctx := v.Context()
		cases = append(cases, reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(ctx.Done()),
		})
		channelOffset = 1

		// Channel recv cases
		for i := 1; i <= channelCount; i++ {
			ch := extractChannel(v.Get(i))
			if ch == nil {
				panic("bad argument to 'chan.select' (channel expected)")
			}
			if ch.Provider() != provider {
				panic("bad argument to 'chan.select' (channel from different VM)")
			}
			cases = append(cases, reflect.SelectCase{
				Dir:  reflect.SelectRecv,
				Chan: reflect.ValueOf(ch.GoChan()),
			})
		}

		// Optional timeout case
		timeoutIdx := -1
		if hasTimeout {
			timeoutIdx = len(cases)
			cases = append(cases, reflect.SelectCase{
				Dir:  reflect.SelectRecv,
				Chan: reflect.ValueOf(time.After(timeout)),
			})
		}

		chosen, recvVal, recvOk := reflect.Select(cases)

		// Post-wake checkpoint
		if err := v.CheckInterrupt(); err != nil {
			panic("interrupted")
		}

		// Context cancelled
		if chosen == 0 {
			panic("interrupted")
		}

		// Timeout
		if hasTimeout && chosen == timeoutIdx {
			v.Set(0, vm.NewInt(0))
			v.Set(1, vm.Nil)
			v.Set(2, vm.False)
			return 3
		}

		// Channel received
		idx := chosen - channelOffset + 1 // 1-indexed Lua
		var val vm.Value
		if recvOk && recvVal.IsValid() {
			val = recvVal.Interface().(vm.Value)
		} else {
			val = vm.Nil
		}

		v.Set(0, vm.NewInt(int64(idx)))
		v.Set(1, val)
		v.Set(2, vm.NewBool(recvOk))
		return 3
	}
}
