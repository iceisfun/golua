package golua_test

// Regression for VM.Close reaping suspended coroutines' goroutines (backport).
// golua backs each coroutine with a goroutine; an abandoned *suspended*
// coroutine parks one that Go cannot reap. VM.Close force-terminates all
// still-suspended coroutines so the leak is bounded to the VM's lifetime.

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
	"github.com/iceisfun/golua/stdlib"
	"github.com/iceisfun/golua/vm"
)

func goroutinesSettled() int {
	runtime.GC()
	time.Sleep(150 * time.Millisecond)
	runtime.GC()
	return runtime.NumGoroutine()
}

func itoaReap(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

func TestVMCloseReapsCoroutines(t *testing.T) {
	const N = 500
	base := goroutinesSettled()

	v := vm.New()
	stdlib.Open(v)
	block, err := parser.Parse("=reap", "for i=1,"+itoaReap(N)+
		" do coroutine.resume(coroutine.create(function() for j=1,50 do coroutine.yield(j) end end)) end")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	proto, err := compiler.Compile("=reap", block)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if _, err := v.Run(proto); err != nil {
		t.Fatalf("run: %v", err)
	}

	if leaked := goroutinesSettled() - base; leaked < N/2 {
		t.Fatalf("expected ~%d suspended-coroutine goroutines before Close, got %d", N, leaked)
	}

	if err := v.Close(context.Background()); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if remaining := goroutinesSettled() - base; remaining > 50 {
		t.Fatalf("VM.Close did not reap suspended-coroutine goroutines: %d still live", remaining)
	}
}
