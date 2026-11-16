package vm

import "testing"

// A thread is backed by a *Table whose own metatable is always nil; its
// metatable is the shared thread type metatable. indexValueDepth and the
// SetIndexValue/SetIndexInt/newIndexValue fallbacks took the "is a table" branch
// for threads and so used the backing table, which made library indexing
// (IndexInt, used by ipairs/table.unpack/table.concat) and __index/__newindex
// chains that pass through a thread miss the thread metatable entirely.
func TestThreadTypeMetatableReachesLibraryIndexing(t *testing.T) {
	v := New()
	threadTbl := NewEmptyTable()
	threadTbl.SetThread(true)
	co := NewTable(threadTbl)

	readFrom := NewEmptyTable()
	readFrom.MustSet(NewInt(1), NewString("one"))
	readFrom.MustSet(NewString("x"), NewString("ex"))
	writeTo := NewEmptyTable()
	tmt := NewEmptyTable()
	tmt.SetString(MetaIndex, NewTable(readFrom))
	tmt.SetString(MetaNewIndex, NewTable(writeTo))
	v.SetTypeMeta(co, tmt)

	if got, err := v.IndexInt(co, 1); err != nil || got.AsString() != "one" {
		t.Fatalf("IndexInt(thread, 1) = %v, %v; want \"one\"", got, err)
	}

	// Userdata whose __index / __newindex is the thread.
	umt := NewEmptyTable()
	umt.SetString(MetaIndex, co)
	umt.SetString(MetaNewIndex, co)
	ud := NewUserdataValue(nil, umt)
	if got, err := v.IndexValue(ud, NewString("x")); err != nil || got.AsString() != "ex" {
		t.Fatalf("userdata __index = thread: got %v, %v; want \"ex\"", got, err)
	}
	if err := v.SetIndexInt(ud, 7, NewString("seven")); err != nil {
		t.Fatalf("SetIndexInt through __newindex = thread: %v", err)
	}
	if err := v.SetIndexValue(ud, NewString("k"), NewString("v")); err != nil {
		t.Fatalf("SetIndexValue through __newindex = thread: %v", err)
	}
	if writeTo.GetInt(7).AsString() != "seven" || writeTo.GetString("k").AsString() != "v" {
		t.Errorf("writes did not reach the thread metatable's __newindex table")
	}
	if !threadTbl.GetInt(7).IsNil() || !threadTbl.GetString("k").IsNil() {
		t.Errorf("writes landed in the thread's backing table")
	}

	// A table whose __index is the thread chains the same way.
	viaTbl := NewEmptyTable()
	vmt := NewEmptyTable()
	vmt.SetString(MetaIndex, co)
	viaTbl.SetMetatable(vmt)
	if got, err := v.IndexValue(NewTable(viaTbl), NewString("x")); err != nil || got.AsString() != "ex" {
		t.Fatalf("table __index = thread: got %v, %v; want \"ex\"", got, err)
	}
}
