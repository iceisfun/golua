package vm

import (
	"strings"
	"testing"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
)

func TestMetatableCycleIndex(t *testing.T) {
	// Create two tables with __index pointing at each other
	t1 := NewEmptyTable()
	t2 := NewEmptyTable()
	mt1 := NewEmptyTable()
	mt2 := NewEmptyTable()
	mt1.Set(metaIndex, NewTable(t2))
	mt2.Set(metaIndex, NewTable(t1))
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
	mt1.Set(metaNewIndex, NewTable(t2))
	mt2.Set(metaNewIndex, NewTable(t1))
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
	mt1.Set(metaIndex, NewTable(t2))
	mt2.Set(metaIndex, NewTable(t1))
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

func TestMetatableValidChain(t *testing.T) {
	// 10-table __index chain (no cycle) should work correctly
	tables := make([]LuaTable, 10)
	for i := range tables {
		tables[i] = NewEmptyTable()
	}
	// Put the value in the last table
	tables[9].Set(NewString("found"), NewInt(99))

	// Chain: tables[0] -> tables[1] -> ... -> tables[9]
	for i := 0; i < 9; i++ {
		mt := NewEmptyTable()
		mt.Set(metaIndex, NewTable(tables[i+1]))
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
