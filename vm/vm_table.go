package vm

import "fmt"

// valueIsThread reports whether v is a coroutine thread. Threads are backed by
// *Table (isThread=true) so Value.IsTable() returns true for them; the __index
// chain-follow must exclude threads and treat them as non-table values, matching
// reference Lua which raises "attempt to index a thread value".
func valueIsThread(v Value) bool {
	if v.IsTable() {
		if tbl, ok := v.AsTable().(*Table); ok {
			return tbl.IsThread()
		}
	}
	return false
}

// tableGet gets a value from a table, handling __index metamethod.
// The loop structure matches Lua's luaV_finishget: the initial table check
// is "free" (not counted), and each loop iteration performs one redirect + check.
// This means MAXTAGLOOP redirects are allowed when the value is found, but
// MAXTAGLOOP redirects with no value triggers the chain-too-long error.
func (vm *VM) tableGet(t LuaTable, key Value) (Value, error) {
	// Fast path: concrete table with no metatable (most common case)
	if ct, ok := t.(*Table); ok && ct.metatable == nil {
		return ct.Get(key), nil
	}
	return vm.tableGetDepth(t, key, vm.MaxMetaDepth())
}

// tableGetDepth is tableGet with an explicit budget of remaining chain hops.
// luaV_finishget walks the whole chain under a single flat counter, so a chain
// that alternates between tables and non-table values has to drain one shared
// budget rather than restart one per hop; every function in this family
// therefore passes the remaining count on instead of re-seeding MaxMetaDepth.
func (vm *VM) tableGetDepth(t LuaTable, key Value, depth int) (Value, error) {
	// Fast path: concrete table with no metatable (most common case)
	if ct, ok := t.(*Table); ok && ct.metatable == nil && depth > 0 {
		return ct.Get(key), nil
	}
	// Initial table check (free, like the inline bytecode fast path)
	val := t.Get(key)
	if !val.IsNil() {
		return val, nil
	}
	for ; depth > 0; depth-- {
		// Key not found, check for __index metamethod
		mt := t.Metatable()
		if mt == nil {
			return Nil, nil
		}

		index := mt.Get(metaIndex)
		if index.IsNil() {
			return Nil, nil
		}

		if index.IsTable() && !valueIsThread(index) {
			// __index is a table, follow the chain
			t = index.AsTable()
			// Check the redirected-to table
			val = t.Get(key)
			if !val.IsNil() {
				return val, nil
			}
			continue
		}

		if index.IsFunction() || index.IsNativeFunc() {
			// __index is a function, call it with (table, key)
			return vm.callMetamethod("index", index, NewTable(t), key)
		}

		// __index is another value — chain through its metatable
		return vm.indexValueDepth(index, key, depth-1)
	}
	return Nil, vm.runtimeError("'__index' chain too long; possible loop")
}

// tableGetString gets a value from a table by string key, handling __index metamethod
func (vm *VM) tableGetString(t LuaTable, key string) (Value, error) {
	// Fast path: concrete table with no metatable
	if ct, ok := t.(*Table); ok && ct.metatable == nil {
		return ct.GetString(key), nil
	}
	return vm.tableGetStringDepth(t, key, vm.MaxMetaDepth())
}

// tableGetStringDepth is tableGetString carrying the shared chain budget.
func (vm *VM) tableGetStringDepth(t LuaTable, key string, depth int) (Value, error) {
	// Fast path: concrete table with no metatable
	if ct, ok := t.(*Table); ok && ct.metatable == nil && depth > 0 {
		return ct.GetString(key), nil
	}
	// Initial table check (free, like the inline bytecode fast path)
	if ct, ok := t.(*Table); ok {
		if val := ct.GetString(key); !val.IsNil() {
			return val, nil
		}
	} else {
		if val := t.Get(NewString(key)); !val.IsNil() {
			return val, nil
		}
	}
	for ; depth > 0; depth-- {
		// Key not found, check for __index metamethod
		mt := t.Metatable()
		if mt == nil {
			return Nil, nil
		}

		index := mt.Get(metaIndex)
		if index.IsNil() {
			return Nil, nil
		}

		if index.IsTable() && !valueIsThread(index) {
			// __index is a table, follow the chain
			t = index.AsTable()
			// Check the redirected-to table
			if ct, ok := t.(*Table); ok {
				if val := ct.GetString(key); !val.IsNil() {
					return val, nil
				}
			} else {
				if val := t.Get(NewString(key)); !val.IsNil() {
					return val, nil
				}
			}
			continue
		}

		if index.IsFunction() || index.IsNativeFunc() {
			// __index is a function, call it with (table, key)
			return vm.callMetamethod("index", index, NewTable(t), NewString(key))
		}

		// __index is another value — chain through its metatable
		return vm.indexValueDepth(index, NewString(key), depth-1)
	}
	return Nil, vm.runtimeError("'__index' chain too long; possible loop")
}

// tableGetInt gets a value from a table by int key, handling __index metamethod
func (vm *VM) tableGetInt(t LuaTable, key int) (Value, error) {
	// Fast path: concrete table with no metatable
	if ct, ok := t.(*Table); ok && ct.metatable == nil {
		return ct.GetInt(key), nil
	}
	return vm.tableGetIntDepth(t, key, vm.MaxMetaDepth())
}

// tableGetIntDepth is tableGetInt carrying the shared chain budget.
func (vm *VM) tableGetIntDepth(t LuaTable, key int, depth int) (Value, error) {
	// Fast path: concrete table with no metatable
	if ct, ok := t.(*Table); ok && ct.metatable == nil && depth > 0 {
		return ct.GetInt(key), nil
	}
	// Initial table check (free, like the inline bytecode fast path)
	if ct, ok := t.(*Table); ok {
		if val := ct.GetInt(key); !val.IsNil() {
			return val, nil
		}
	} else {
		if val := t.Get(NewInt(int64(key))); !val.IsNil() {
			return val, nil
		}
	}
	for ; depth > 0; depth-- {
		// Key not found, check for __index metamethod
		mt := t.Metatable()
		if mt == nil {
			return Nil, nil
		}

		index := mt.Get(metaIndex)
		if index.IsNil() {
			return Nil, nil
		}

		if index.IsTable() && !valueIsThread(index) {
			// __index is a table, follow the chain
			t = index.AsTable()
			// Check the redirected-to table
			if ct, ok := t.(*Table); ok {
				if val := ct.GetInt(key); !val.IsNil() {
					return val, nil
				}
			} else {
				if val := t.Get(NewInt(int64(key))); !val.IsNil() {
					return val, nil
				}
			}
			continue
		}

		if index.IsFunction() || index.IsNativeFunc() {
			// __index is a function, call it with (table, key)
			return vm.callMetamethod("index", index, NewTable(t), NewInt(int64(key)))
		}

		// __index is another value — chain through its metatable
		return vm.indexValueDepth(index, NewInt(int64(key)), depth-1)
	}
	return Nil, vm.runtimeError("'__index' chain too long; possible loop")
}

// indexValue tries to index a non-table value by looking up its metatable.
// For example, if __index is a string, this chains through the string metatable.
func (vm *VM) indexValue(val Value, key Value) (Value, error) {
	return vm.indexValueDepth(val, key, vm.MaxMetaDepth())
}

func (vm *VM) indexValueDepth(val Value, key Value, depth int) (Value, error) {
	if depth <= 0 {
		return Nil, vm.runtimeError("'__index' chain too long; possible loop")
	}
	// Get the metatable for this value type
	var mt LuaTable
	if val.IsTable() {
		mt = val.AsTable().Metatable()
	} else if ud := val.AsUserdata(); ud != nil {
		mt = ud.Metatable()
	} else {
		mt = vm.GetTypeMeta(val)
	}
	if mt == nil {
		return Nil, vm.runtimeError("attempt to index a %s value", vm.ObjTypeName(val))
	}
	index := mt.Get(metaIndex)
	if index.IsNil() {
		return Nil, vm.runtimeError("attempt to index a %s value", vm.ObjTypeName(val))
	}
	if index.IsTable() {
		return vm.tableGetDepth(index.AsTable(), key, depth-1)
	}
	if index.IsFunction() || index.IsNativeFunc() {
		return vm.callMetamethod("index", index, val, key)
	}
	// __index is another value (e.g. a string) — chain through its metatable
	return vm.indexValueDepth(index, key, depth-1)
}

// resolveIndex resolves an __index metamethod for a non-table value.
// mm is the __index metamethod value, obj is the original value, key is the lookup key.
func (vm *VM) resolveIndex(mm Value, obj Value, key Value) (Value, error) {
	// Reaching mm already consumed one hop of the chain budget (the caller
	// looked up obj's type metatable), so the walk continues one short.
	depth := vm.MaxMetaDepth() - 1
	if mm.IsTable() {
		return vm.tableGetDepth(mm.AsTable(), key, depth)
	}
	if mm.IsFunction() || mm.IsNativeFunc() {
		return vm.callMetamethod("index", mm, obj, key)
	}
	// __index is another value (e.g. a string) — chain through its metatable
	return vm.indexValueDepth(mm, key, depth)
}

// tableSet sets a value in a table, handling __newindex metamethod.
// The loop structure matches Lua's luaV_finishset: the initial table check
// is "free" (not counted), and each loop iteration performs one redirect + check.
func (vm *VM) tableSet(t LuaTable, key, value Value) error {
	// Fast path: concrete table with no metatable (skip existence check)
	if ct, ok := t.(*Table); ok && ct.metatable == nil {
		if err := ct.Set(key, value); err != nil {
			return vm.runtimeError("%s", err)
		}
		return nil
	}
	return vm.tableSetDepth(t, key, value, vm.MaxMetaDepth())
}

// tableSetDepth is tableSet with an explicit budget of remaining chain hops,
// shared with newIndexValue so a chain that alternates between tables and
// non-table values drains one counter, as luaV_finishset's flat loop does.
func (vm *VM) tableSetDepth(t LuaTable, key, value Value, depth int) error {
	// Fast path: concrete table with no metatable (skip existence check)
	if ct, ok := t.(*Table); ok && ct.metatable == nil && depth > 0 {
		if err := ct.Set(key, value); err != nil {
			return vm.runtimeError("%s", err)
		}
		return nil
	}
	// Initial check (free, like the inline bytecode fast path)
	existing := t.Get(key)
	if !existing.IsNil() {
		if err := t.Set(key, value); err != nil {
			return vm.runtimeError("%s", err)
		}
		return nil
	}
	for ; depth > 0; depth-- {
		// Key doesn't exist, check for __newindex metamethod
		mt := t.Metatable()
		if mt == nil {
			if err := t.Set(key, value); err != nil {
				return vm.runtimeError("%s", err)
			}
			return nil
		}

		newindex := mt.Get(metaNewIndex)
		if newindex.IsNil() {
			if err := t.Set(key, value); err != nil {
				return vm.runtimeError("%s", err)
			}
			return nil
		}

		if newindex.IsTable() {
			if tbl, ok := newindex.AsTable().(*Table); ok && tbl.IsThread() {
				return vm.newIndexValue(newindex, key, value, depth-1)
			}
			// __newindex is a table, follow the chain
			t = newindex.AsTable()
			// Check the redirected-to table
			existing = t.Get(key)
			if !existing.IsNil() {
				if err := t.Set(key, value); err != nil {
					return vm.runtimeError("%s", err)
				}
				return nil
			}
			continue
		}

		if newindex.IsFunction() || newindex.IsNativeFunc() {
			// __newindex is a function, call it with (table, key, value)
			_, err := vm.callMetamethod3("newindex", newindex, NewTable(t), key, value)
			return err
		}

		// __newindex is a non-table, non-function value — chain through its metatable
		return vm.newIndexValue(newindex, key, value, depth-1)
	}
	return vm.runtimeError("'__newindex' chain too long; possible loop")
}

// tableSetString sets a value in a table by string key, handling __newindex metamethod
func (vm *VM) tableSetString(t LuaTable, key string, value Value) error {
	// Fast path: concrete table with no metatable
	if ct, ok := t.(*Table); ok && ct.metatable == nil {
		ct.SetString(key, value)
		return nil
	}
	return vm.tableSetStringDepth(t, key, value, vm.MaxMetaDepth())
}

// tableSetStringDepth is tableSetString carrying the shared chain budget.
func (vm *VM) tableSetStringDepth(t LuaTable, key string, value Value, depth int) error {
	// Fast path: concrete table with no metatable
	if ct, ok := t.(*Table); ok && ct.metatable == nil && depth > 0 {
		ct.SetString(key, value)
		return nil
	}
	// Initial check (free, like the inline bytecode fast path)
	if ct, ok := t.(*Table); ok {
		if existing := ct.GetString(key); !existing.IsNil() {
			ct.SetString(key, value)
			return nil
		}
	} else {
		keyVal := NewString(key)
		if existing := t.Get(keyVal); !existing.IsNil() {
			if err := t.Set(keyVal, value); err != nil {
				return err
			}
			return nil
		}
	}
	for ; depth > 0; depth-- {
		// Fast path: use *Table methods to avoid NewString allocation
		if ct, ok := t.(*Table); ok {
			mt := ct.Metatable()
			if mt == nil {
				ct.SetString(key, value)
				return nil
			}

			newindex := mt.Get(metaNewIndex)
			if newindex.IsNil() {
				ct.SetString(key, value)
				return nil
			}

			if newindex.IsTable() {
				if tbl, ok := newindex.AsTable().(*Table); ok && tbl.IsThread() {
					// Thread: chain through type metatable, not the thread table itself
					return vm.newIndexValue(newindex, NewString(key), value, depth-1)
				}
				t = newindex.AsTable()
				// Check the redirected-to table
				if ct2, ok := t.(*Table); ok {
					if existing := ct2.GetString(key); !existing.IsNil() {
						ct2.SetString(key, value)
						return nil
					}
				} else {
					keyVal := NewString(key)
					if existing := t.Get(keyVal); !existing.IsNil() {
						if err := t.Set(keyVal, value); err != nil {
							return err
						}
						return nil
					}
				}
				continue
			}

			if newindex.IsFunction() || newindex.IsNativeFunc() {
				_, err := vm.callMetamethod3("newindex", newindex, NewTable(t), NewString(key), value)
				return err
			}

			return vm.newIndexValue(newindex, NewString(key), value, depth-1)
		}

		// Slow path: generic LuaTable interface
		keyVal := NewString(key)
		mt := t.Metatable()
		if mt == nil {
			if err := t.Set(keyVal, value); err != nil {
				return err
			}
			return nil
		}

		newindex := mt.Get(metaNewIndex)
		if newindex.IsNil() {
			if err := t.Set(keyVal, value); err != nil {
				return err
			}
			return nil
		}

		if newindex.IsTable() {
			if tbl, ok := newindex.AsTable().(*Table); ok && tbl.IsThread() {
				return vm.newIndexValue(newindex, keyVal, value, depth-1)
			}
			t = newindex.AsTable()
			// Check the redirected-to table
			if existing := t.Get(keyVal); !existing.IsNil() {
				if err := t.Set(keyVal, value); err != nil {
					return err
				}
				return nil
			}
			continue
		}

		if newindex.IsFunction() || newindex.IsNativeFunc() {
			_, err := vm.callMetamethod3("newindex", newindex, NewTable(t), keyVal, value)
			return err
		}

		return vm.newIndexValue(newindex, keyVal, value, depth-1)
	}
	return vm.runtimeError("'__newindex' chain too long; possible loop")
}

// newIndexValue handles __newindex assignment on a non-table value by looking up
// its metatable's __newindex and chaining through it.
func (vm *VM) newIndexValue(val Value, key, value Value, depth int) error {
	if depth <= 0 {
		return vm.runtimeError("'__newindex' chain too long; possible loop")
	}
	mt := vm.GetTypeMeta(val)
	if ud := val.AsUserdata(); ud != nil {
		mt = ud.Metatable()
	}
	if mt == nil {
		return vm.runtimeError("attempt to index a %s value", vm.ObjTypeName(val))
	}
	newindex := mt.Get(metaNewIndex)
	if newindex.IsNil() {
		return vm.runtimeError("attempt to index a %s value", vm.ObjTypeName(val))
	}
	if newindex.IsTable() {
		return vm.tableSetDepth(newindex.AsTable(), key, value, depth-1)
	}
	if newindex.IsFunction() || newindex.IsNativeFunc() {
		_, err := vm.callMetamethod3("newindex", newindex, val, key, value)
		return err
	}
	return vm.newIndexValue(newindex, key, value, depth-1)
}

// tableSetInt sets a value in a table by int key, handling __newindex metamethod
func (vm *VM) tableSetInt(t LuaTable, key int, value Value) error {
	// Fast path: concrete table with no metatable
	if ct, ok := t.(*Table); ok && ct.metatable == nil {
		ct.SetInt(key, value)
		return nil
	}
	return vm.tableSetIntDepth(t, key, value, vm.MaxMetaDepth())
}

// tableSetIntDepth is tableSetInt carrying the shared chain budget.
func (vm *VM) tableSetIntDepth(t LuaTable, key int, value Value, depth int) error {
	// Fast path: concrete table with no metatable
	if ct, ok := t.(*Table); ok && ct.metatable == nil && depth > 0 {
		ct.SetInt(key, value)
		return nil
	}
	// Initial check (free, like the inline bytecode fast path)
	if ct, ok := t.(*Table); ok {
		if existing := ct.GetInt(key); !existing.IsNil() {
			ct.SetInt(key, value)
			return nil
		}
	} else {
		keyVal := NewInt(int64(key))
		if existing := t.Get(keyVal); !existing.IsNil() {
			if err := t.Set(keyVal, value); err != nil {
				return err
			}
			return nil
		}
	}
	for ; depth > 0; depth-- {
		// Fast path: use *Table methods to avoid NewInt/hashKey overhead
		if ct, ok := t.(*Table); ok {
			mt := ct.Metatable()
			if mt == nil {
				ct.SetInt(key, value)
				return nil
			}

			newindex := mt.Get(metaNewIndex)
			if newindex.IsNil() {
				ct.SetInt(key, value)
				return nil
			}

			if newindex.IsTable() {
				if tbl, ok := newindex.AsTable().(*Table); ok && tbl.IsThread() {
					return vm.newIndexValue(newindex, NewInt(int64(key)), value, depth-1)
				}
				t = newindex.AsTable()
				// Check the redirected-to table
				if ct2, ok := t.(*Table); ok {
					if existing := ct2.GetInt(key); !existing.IsNil() {
						ct2.SetInt(key, value)
						return nil
					}
				} else {
					keyVal := NewInt(int64(key))
					if existing := t.Get(keyVal); !existing.IsNil() {
						if err := t.Set(keyVal, value); err != nil {
							return err
						}
						return nil
					}
				}
				continue
			}

			if newindex.IsFunction() || newindex.IsNativeFunc() {
				_, err := vm.callMetamethod3("newindex", newindex, NewTable(t), NewInt(int64(key)), value)
				return err
			}

			return vm.newIndexValue(newindex, NewInt(int64(key)), value, depth-1)
		}

		// Slow path: generic LuaTable interface
		keyVal := NewInt(int64(key))
		mt := t.Metatable()
		if mt == nil {
			if err := t.Set(keyVal, value); err != nil {
				return err
			}
			return nil
		}

		newindex := mt.Get(metaNewIndex)
		if newindex.IsNil() {
			if err := t.Set(keyVal, value); err != nil {
				return err
			}
			return nil
		}

		if newindex.IsTable() {
			if tbl, ok := newindex.AsTable().(*Table); ok && tbl.IsThread() {
				return vm.newIndexValue(newindex, keyVal, value, depth-1)
			}
			t = newindex.AsTable()
			// Check the redirected-to table
			if existing := t.Get(keyVal); !existing.IsNil() {
				if err := t.Set(keyVal, value); err != nil {
					return err
				}
				return nil
			}
			continue
		}

		if newindex.IsFunction() || newindex.IsNativeFunc() {
			_, err := vm.callMetamethod3("newindex", newindex, NewTable(t), keyVal, value)
			return err
		}

		return vm.newIndexValue(newindex, keyVal, value, depth-1)
	}
	return vm.runtimeError("'__newindex' chain too long; possible loop")
}

// TableGet retrieves t[key] with __index metamethod support.
func (vm *VM) TableGet(t LuaTable, key Value) (Value, error) {
	return vm.tableGet(t, key)
}

// TableGetInt retrieves t[key] with __index metamethod support.
func (vm *VM) TableGetInt(t LuaTable, key int) (Value, error) {
	return vm.tableGetInt(t, key)
}

// SetIndexValue sets val[key]=value with __newindex metamethod support.
func (vm *VM) SetIndexValue(val Value, key Value, value Value) error {
	if ct, ok := val.ptr.(*Table); ok && val.typ == typeTable && !ct.isThread {
		if ct.metatable == nil {
			if err := ct.Set(key, value); err != nil {
				return vm.runtimeError("%s", err)
			}
			return nil
		}
		return vm.tableSet(val.ptr.(LuaTable), key, value)
	}
	if mm := vm.getMetafield(val, MetaNewIndex); !mm.IsNil() {
		if mm.IsFunction() || mm.IsNativeFunc() {
			_, err := vm.callMetamethod3("newindex", mm, val, key, value)
			return err
		}
		if mm.IsTable() {
			// The metafield lookup above already consumed one hop.
			return vm.tableSetDepth(mm.AsTable(), key, value, vm.MaxMetaDepth()-1)
		}
		return vm.runtimeError("attempt to index a %s value", vm.ObjTypeName(mm))
	}
	return vm.runtimeError("attempt to index a %s value", vm.ObjTypeName(val))
}

// SetIndexInt sets val[key]=value with __newindex metamethod support.
func (vm *VM) SetIndexInt(val Value, key int, value Value) error {
	if ct, ok := val.ptr.(*Table); ok && val.typ == typeTable && !ct.isThread {
		if ct.metatable == nil {
			ct.SetInt(key, value)
			return nil
		}
		return vm.tableSetInt(val.ptr.(LuaTable), key, value)
	}
	if mm := vm.getMetafield(val, MetaNewIndex); !mm.IsNil() {
		if mm.IsFunction() || mm.IsNativeFunc() {
			_, err := vm.callMetamethod3("newindex", mm, val, NewInt(int64(key)), value)
			return err
		}
		if mm.IsTable() {
			// The metafield lookup above already consumed one hop.
			return vm.tableSetDepth(mm.AsTable(), NewInt(int64(key)), value, vm.MaxMetaDepth()-1)
		}
		return vm.runtimeError("attempt to index a %s value", vm.ObjTypeName(mm))
	}
	return vm.runtimeError("attempt to index a %s value", vm.ObjTypeName(val))
}

// TableSetInt sets t[key]=value with __newindex metamethod support.
func (vm *VM) TableSetInt(t LuaTable, key int, value Value) error {
	return vm.tableSetInt(t, key, value)
}

// IndexValue retrieves val[key] with __index metamethod support for any value type.
func (vm *VM) IndexValue(val Value, key Value) (Value, error) {
	// Fast path: concrete *Table with no metatable (mirrors SetIndexValue and
	// the GETTABLE opcode), avoiding the AsTable/tableGet interface round-trip.
	if ct, ok := val.ptr.(*Table); ok && val.typ == typeTable && !ct.isThread && ct.metatable == nil {
		return ct.Get(key), nil
	}
	if val.IsTable() && !val.AsTable().IsThread() {
		return vm.tableGet(val.AsTable(), key)
	}
	return vm.indexValue(val, key)
}

// IndexInt retrieves val[key] for an integer key with __index metamethod
// support. It mirrors SetIndexInt: the fast path reads directly from a
// metatable-free concrete *Table via GetInt, avoiding boxing the key into a
// Value and the IsNumber/ToInt decode that IndexValue+Table.Get would perform.
func (vm *VM) IndexInt(val Value, key int) (Value, error) {
	if ct, ok := val.ptr.(*Table); ok && val.typ == typeTable && !ct.isThread && ct.metatable == nil {
		return ct.GetInt(key), nil
	}
	if val.IsTable() && !val.AsTable().IsThread() {
		return vm.tableGetInt(val.AsTable(), key)
	}
	return vm.indexValue(val, NewInt(int64(key)))
}

// ObjLen returns #v with __len metamethod support.
func (vm *VM) ObjLen(val Value) (int, error) {
	if val.IsString() {
		return len(val.AsString()), nil
	}
	mm := vm.getMetafield(val, MetaLen)
	if !mm.IsNil() {
		// Reference lvm.c passes the object twice (luaT_callTMres(L, tm, rb, rb, ra)),
		// so __len must see the same second argument here as it does from OP_LEN.
		res, err := vm.callMetamethod("len", mm, val, val)
		if err != nil {
			return 0, err
		}
		if i, ok := res.ToInt(); ok {
			return int(i), nil
		}
		return 0, fmt.Errorf("object length is not an integer")
	}
	if val.IsTable() && !val.AsTable().IsThread() {
		return val.AsTable().Len(), nil
	}
	return 0, vm.runtimeError("attempt to get length of a %s value", vm.ObjTypeName(val))
}
