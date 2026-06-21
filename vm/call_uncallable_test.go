package vm

import (
	"strings"
	"testing"
)

// TestCallUncallableReportsResolvedType verifies that when a value is called via
// the C-boundary path (ProtectedCall / pcall(t)) and its __call chain dead-ends
// on a non-callable value, the "attempt to call a X value" error reports the
// type of the value that actually failed to be callable (the resolved __call
// metavalue), not the type of the original object. This matches reference Lua
// 5.5 and the OP_CALL (named-call) path, which were already correct.
func TestCallUncallableReportsResolvedType(t *testing.T) {
	cases := []struct {
		name     string
		build    func() Value
		wantType string
	}{
		{
			name: "__call is a number",
			build: func() Value {
				tbl := NewEmptyTable()
				mt := NewEmptyTable()
				mt.MustSet(NewString("__call"), NewInt(5))
				tbl.SetMetatable(mt)
				return NewTable(tbl)
			},
			wantType: "number",
		},
		{
			name: "__call is a boolean",
			build: func() Value {
				tbl := NewEmptyTable()
				mt := NewEmptyTable()
				mt.MustSet(NewString("__call"), True)
				tbl.SetMetatable(mt)
				return NewTable(tbl)
			},
			wantType: "boolean",
		},
		{
			name: "__call chain dead-ends on a string",
			build: func() Value {
				inner := NewEmptyTable()
				innerMt := NewEmptyTable()
				innerMt.MustSet(NewString("__call"), NewString("x"))
				inner.SetMetatable(innerMt)

				outer := NewEmptyTable()
				outerMt := NewEmptyTable()
				outerMt.MustSet(NewString("__call"), NewTable(inner))
				outer.SetMetatable(outerMt)
				return NewTable(outer)
			},
			wantType: "string",
		},
		{
			name: "no __call at all reports the object type",
			build: func() Value {
				tbl := NewEmptyTable()
				tbl.SetMetatable(NewEmptyTable())
				return NewTable(tbl)
			},
			wantType: "table",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v := New()
			_, err := v.ProtectedCall(tc.build(), nil)
			if err == nil {
				t.Fatalf("expected an error calling an uncallable value, got nil")
			}
			want := "attempt to call a " + tc.wantType + " value"
			if !strings.Contains(err.Error(), want) {
				t.Errorf("expected error to contain %q, got: %s", want, err.Error())
			}
		})
	}
}
