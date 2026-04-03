package vm

import (
	"strings"
	"testing"

	"github.com/iceisfun/golua/v2/compiler"
	"github.com/iceisfun/golua/v2/parser"
)

func TestMetatableCycleIndex(t *testing.T) {
	// Create two tables with __index pointing at each other
	t1 := NewEmptyTable()
	t2 := NewEmptyTable()
	mt1 := NewEmptyTable()
	mt2 := NewEmptyTable()
	mt1.MustSet(metaIndex, NewTable(t2))
	mt2.MustSet(metaIndex, NewTable(t1))
	t1.SetMetatable(mt1)
	t2.SetMetatable(mt2)

	v := New()
	_, err := v.tableGet(t1, NewString("missing"))
	if err == nil {
		t.Fatal("expected error from __index cycle")
	}
	if !strings.Contains(err.Error(), "'__index' chain too long; possible loop") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMetatableCycleNewIndex(t *testing.T) {
	// Create two tables with __newindex pointing at each other
	t1 := NewEmptyTable()
	t2 := NewEmptyTable()
	mt1 := NewEmptyTable()
	mt2 := NewEmptyTable()
	mt1.MustSet(metaNewIndex, NewTable(t2))
	mt2.MustSet(metaNewIndex, NewTable(t1))
	t1.SetMetatable(mt1)
	t2.SetMetatable(mt2)

	v := New()
	err := v.tableSet(t1, NewString("key"), NewInt(42))
	if err == nil {
		t.Fatal("expected error from __newindex cycle")
	}
	if !strings.Contains(err.Error(), "'__newindex' chain too long; possible loop") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMetatableCycleIndexLua(t *testing.T) {
	// Set up cycle in Go and inject tables, then access in Lua
	t1 := NewEmptyTable()
	t2 := NewEmptyTable()
	mt1 := NewEmptyTable()
	mt2 := NewEmptyTable()
	mt1.MustSet(metaIndex, NewTable(t2))
	mt2.MustSet(metaIndex, NewTable(t1))
	t1.SetMetatable(mt1)
	t2.SetMetatable(mt2)

	source := `return t1.missing`
	block, err := parser.Parse("<test>", source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto, err := compiler.Compile("<test>", block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	v := New()
	v.SetGlobal("t1", NewTable(t1))
	_, err = v.Run(proto)
	if err == nil {
		t.Fatal("expected error from __index cycle in Lua")
	}
	if !strings.Contains(err.Error(), "__index") {
		t.Fatalf("expected __index error, got: %v", err)
	}
}

func TestMaxMetaDepthDefault(t *testing.T) {
	// Default VM (no options) should use DefaultMaxMetaDepth
	v := New()
	if v.MaxMetaDepth() != DefaultMaxMetaDepth {
		t.Fatalf("expected default %d, got %d", DefaultMaxMetaDepth, v.MaxMetaDepth())
	}
}

func TestMaxMetaDepthZeroMeansDefault(t *testing.T) {
	// MaxMetaDepth = 0 in Limits should use the default
	v := New(WithLimits(Limits{MaxMetaDepth: 0}))
	if v.MaxMetaDepth() != DefaultMaxMetaDepth {
		t.Fatalf("expected default %d, got %d", DefaultMaxMetaDepth, v.MaxMetaDepth())
	}
}

func TestMaxMetaDepthCustomLower(t *testing.T) {
	// Custom lower limit triggers earlier error
	v := New(WithMaxMetaDepth(5))
	if v.MaxMetaDepth() != 5 {
		t.Fatalf("expected 5, got %d", v.MaxMetaDepth())
	}

	// Build a cycle and verify it errors within 5 steps
	t1 := NewEmptyTable()
	t2 := NewEmptyTable()
	mt1 := NewEmptyTable()
	mt2 := NewEmptyTable()
	mt1.MustSet(metaIndex, NewTable(t2))
	mt2.MustSet(metaIndex, NewTable(t1))
	t1.SetMetatable(mt1)
	t2.SetMetatable(mt2)

	_, err := v.tableGet(t1, NewString("missing"))
	if err == nil {
		t.Fatal("expected error from __index cycle with custom limit")
	}
	if !strings.Contains(err.Error(), "'__index' chain too long; possible loop") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMaxMetaDepthCustomLowerNewIndex(t *testing.T) {
	v := New(WithMaxMetaDepth(5))

	t1 := NewEmptyTable()
	t2 := NewEmptyTable()
	mt1 := NewEmptyTable()
	mt2 := NewEmptyTable()
	mt1.MustSet(metaNewIndex, NewTable(t2))
	mt2.MustSet(metaNewIndex, NewTable(t1))
	t1.SetMetatable(mt1)
	t2.SetMetatable(mt2)

	err := v.tableSet(t1, NewString("key"), NewInt(42))
	if err == nil {
		t.Fatal("expected error from __newindex cycle with custom limit")
	}
	if !strings.Contains(err.Error(), "'__newindex' chain too long; possible loop") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWithMaxMetaDepthOption(t *testing.T) {
	v := New(WithMaxMetaDepth(100))
	if v.MaxMetaDepth() != 100 {
		t.Fatalf("expected 100, got %d", v.MaxMetaDepth())
	}
}

func TestWithMaxMetaDepthNegativeResetsDefault(t *testing.T) {
	v := New(WithMaxMetaDepth(100), WithMaxMetaDepth(-1))
	if v.MaxMetaDepth() != DefaultMaxMetaDepth {
		t.Fatalf("expected default %d after negative reset, got %d", DefaultMaxMetaDepth, v.MaxMetaDepth())
	}
}

func TestSetMaxMetaDepthMethod(t *testing.T) {
	v := New()
	v.SetMaxMetaDepth(500)
	if v.MaxMetaDepth() != 500 {
		t.Fatalf("expected 500, got %d", v.MaxMetaDepth())
	}

	// Reset to default
	v.SetMaxMetaDepth(0)
	if v.MaxMetaDepth() != DefaultMaxMetaDepth {
		t.Fatalf("expected default %d, got %d", DefaultMaxMetaDepth, v.MaxMetaDepth())
	}
}

func TestWithLimitsMaxMetaDepth(t *testing.T) {
	v := New(WithLimits(Limits{MaxMetaDepth: 300}))
	if v.MaxMetaDepth() != 300 {
		t.Fatalf("expected 300, got %d", v.MaxMetaDepth())
	}
}

func TestWithMaxMetaDepthOverridesWithLimits(t *testing.T) {
	// WithMaxMetaDepth applied after WithLimits should override
	v := New(WithLimits(Limits{MaxMetaDepth: 300}), WithMaxMetaDepth(50))
	if v.MaxMetaDepth() != 50 {
		t.Fatalf("expected 50 (override), got %d", v.MaxMetaDepth())
	}
}

func TestMaxMetaDepthValidChainBelowLimit(t *testing.T) {
	// Chain of 3 tables with limit=5 should work fine
	v := New(WithMaxMetaDepth(5))

	t1 := NewEmptyTable()
	t2 := NewEmptyTable()
	t3 := NewEmptyTable()
	t3.MustSet(NewString("found"), NewInt(77))

	mt1 := NewEmptyTable()
	mt1.MustSet(metaIndex, NewTable(t2))
	t1.SetMetatable(mt1)

	mt2 := NewEmptyTable()
	mt2.MustSet(metaIndex, NewTable(t3))
	t2.SetMetatable(mt2)

	val, err := v.tableGet(t1, NewString("found"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val.AsInt() != 77 {
		t.Fatalf("expected 77, got %v", val)
	}
}

func TestMaxMetaDepthFunctionMetamethodUnaffected(t *testing.T) {
	// Function metamethod should still work regardless of depth limit
	v := New(WithMaxMetaDepth(3))

	tbl := NewEmptyTable()
	mt := NewEmptyTable()
	mt.MustSet(metaIndex, NewNativeFunc(func(vm *VM) int {
		vm.Set(0, NewInt(999))
		return 1
	}))
	tbl.SetMetatable(mt)
	v.SetGlobal("tbl", NewTable(tbl))

	source := `return tbl.anything`
	block, err := parser.Parse("<test>", source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto, err := compiler.Compile("<test>", block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	results, err := v.Run(proto)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) == 0 || results[0].AsInt() != 999 {
		t.Fatalf("expected 999, got %v", results)
	}
}

func TestMetatableValidChain(t *testing.T) {
	// 10-table __index chain (no cycle) should work correctly
	tables := make([]LuaTable, 10)
	for i := range tables {
		tables[i] = NewEmptyTable()
	}
	// Put the value in the last table
	_ = tables[9].Set(NewString("found"), NewInt(99))

	// Chain: tables[0] -> tables[1] -> ... -> tables[9]
	for i := 0; i < 9; i++ {
		mt := NewEmptyTable()
		mt.MustSet(metaIndex, NewTable(tables[i+1]))
		tables[i].SetMetatable(mt)
	}

	v := New()
	val, err := v.tableGet(tables[0], NewString("found"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val.AsInt() != 99 {
		t.Fatalf("expected 99, got %v", val)
	}

	// Key that doesn't exist should return nil without error
	val, err = v.tableGet(tables[0], NewString("nope"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !val.IsNil() {
		t.Fatalf("expected nil, got %v", val)
	}
}
