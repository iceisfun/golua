package stdlib

import (
	"reflect"
	"time"

	"github.com/iceisfun/golua/vm"
)

// channelHandleMeta is a shared metatable for identifying channel handles.
var channelHandleMeta *vm.Table

func init() {
	channelHandleMeta = vm.NewEmptyTable()
	channelHandleMeta.SetString("__name", vm.NewString("channel"))
}

// channelRegistry maps channel IDs to LuaChannel pointers.
// Keyed by int64 ID, same pattern as the coroutine registry.
var channelRegistry = make(map[int64]*vm.LuaChannel)

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

// makeChannelHandle creates a Lua table handle wrapping a LuaChannel.
func makeChannelHandle(v *vm.VM, ch *vm.LuaChannel) vm.Value {
	channelRegistry[ch.ID()] = ch

	handle := vm.NewEmptyTable()
	handle.SetMetatable(channelHandleMeta)
	handle.SetString("__chan_id", vm.NewInt(ch.ID()))

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

// extractChannel extracts a *LuaChannel from a Lua channel handle table.
func extractChannel(v vm.Value) *vm.LuaChannel {
	t := v.AsTable()
	if t == nil {
		return nil
	}
	idVal := t.Get(vm.NewString("__chan_id"))
	if idVal.IsNil() {
		return nil
	}
	id := idVal.AsInt()
	return channelRegistry[id]
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
