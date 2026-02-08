package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/iceisfun/golua/vm"
)

func TestTailCallMutualRecursion(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	source := `
		local function flop(n)
			return flip(n + 1)
		end
		function flip(n)
			return flop(n + 1)
		end
		return flip(0)
	`
	_, err := runLuaWithContext(t, source, "test_mutual_tailcall", ctx, vm.Limits{})
	if err == nil {
		t.Fatal("expected error from context timeout on mutual tail call")
	}
	if !strings.Contains(err.Error(), "execution interrupted") {
		t.Fatalf("expected 'execution interrupted', got: %v", err)
	}
}
