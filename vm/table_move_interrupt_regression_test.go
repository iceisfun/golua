package vm_test

// table.move copies without re-entering the VM's instruction dispatch, so
// without an explicit poll a large range ignores context cancellation entirely:
// the host's deadline passes and the call keeps running. Reference Lua accepts
// ranges spanning the whole integer domain (its own suite relies on it), so the
// range cannot be capped up front — the copy loop has to check for itself.

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/iceisfun/golua/vm"
)

func TestTableMoveRespectsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	// A move over a huge range into a table with an __index/__newindex pair
	// keeps the copy loop in the table library for far longer than the
	// deadline.
	src := `
local proxy = setmetatable({}, {
  __index = function() return 1 end,
  __newindex = function() end,
})
table.move(proxy, 1, math.maxinteger // 2, 1, proxy)
`
	v, run := newScriptVM(t, src, vm.WithContext(ctx))
	_ = v

	done := make(chan error, 1)
	go func() { done <- run() }()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected the cancelled context to interrupt table.move")
		}
		if !strings.Contains(err.Error(), "execution interrupted") {
			t.Fatalf("error = %v, want the context cancellation to surface", err)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("table.move ignored the context deadline")
	}
}
