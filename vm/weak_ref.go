package vm

import "weak"

// weakRef holds a weak pointer to a GC-managed Lua object for weak table
// support (__mode). Because weak.Pointer is generic and Go lacks sum types,
// we use separate fields — only one is non-zero at a time.
//
// A zero weakRef indicates a value type (nil, bool, int, float, string,
// lightuserdata) that is never garbage collected. Use isZero() to distinguish
// this from a collected (dead) reference.
type weakRef struct {
	tbl weak.Pointer[Table]
	cls weak.Pointer[Closure]
	nfn weak.Pointer[nativeFuncBox]
	ud  weak.Pointer[Userdata]
}

// makeWeakRef creates a weakRef for the given Value. Returns (ref, true) if
// the value is a collectable type (table, function, full userdata). Returns
// (zero, false) for value types that are never collected.
func makeWeakRef(v Value) (weakRef, bool) {
	switch v.typ {
	case typeTable:
		if t, ok := v.ptr.(*Table); ok && t != nil {
			return weakRef{tbl: weak.Make(t)}, true
		}
	case typeFunction:
		if c, ok := v.ptr.(*Closure); ok && c != nil {
			return weakRef{cls: weak.Make(c)}, true
		}
	case typeNativeFunc:
		if n, ok := v.ptr.(*nativeFuncBox); ok && n != nil {
			return weakRef{nfn: weak.Make(n)}, true
		}
	case typeUpvalue:
		// Full userdata is collectable; light userdata (upvalue IDs) is not.
		if ud, ok := v.ptr.(*Userdata); ok && ud != nil {
			return weakRef{ud: weak.Make(ud)}, true
		}
	}
	return weakRef{}, false
}

// alive reports whether the weakly-held object is still reachable by Go's GC.
// Returns false for dead (collected) references and for zero weakRefs.
// Callers must check isZero() to distinguish value types from collected refs.
func (w weakRef) alive() bool {
	return w.tbl.Value() != nil ||
		w.cls.Value() != nil ||
		w.nfn.Value() != nil ||
		w.ud.Value() != nil
}

// value reconstructs the original Value from the weak pointer, or returns
// (Nil, false) if the object has been collected.
func (w weakRef) value() (Value, bool) {
	if t := w.tbl.Value(); t != nil {
		return NewTable(t), true
	}
	if c := w.cls.Value(); c != nil {
		return NewFunction(c), true
	}
	if n := w.nfn.Value(); n != nil {
		return Value{typ: typeNativeFunc, ptr: n}, true
	}
	if u := w.ud.Value(); u != nil {
		return Value{typ: typeUpvalue, ptr: u}, true
	}
	return Nil, false
}

// isZero reports whether w is the zero value, indicating a non-collectable
// value type that doesn't need weak tracking.
func (w weakRef) isZero() bool {
	return w == weakRef{}
}

// isCollectable reports whether a Value is a GC-collectable type that can
// participate in weak reference tracking. Strings, numbers, booleans, nil,
// and light userdata are never collected per Lua 5.4 §2.5.4.
func isCollectable(v Value) bool {
	switch v.typ {
	case typeTable, typeFunction, typeNativeFunc:
		return true
	case typeUpvalue:
		_, ok := v.ptr.(*Userdata)
		return ok
	default:
		return false
	}
}
