package vm

import "fmt"

// tableGet gets a value from a table, handling __index metamethod.
// The loop structure matches Lua 5.4's luaV_finishget: the initial table check
// is "free" (not counted), and each loop iteration performs one redirect + check.
// This means MAXTAGLOOP redirects are allowed when the value is found, but
// MAXTAGLOOP redirects with no value triggers the chain-too-long error.
func (vm *VM) tableGet(t LuaTable, key Value) (Value, error) {
	// Fast path: concrete table with no metatable (most common case)
	if ct, ok := t.(*Table); ok && ct.metatable == nil {
		return ct.Get(key), nil
	}
	// Initial table check (free, like Lua 5.4's inline bytecode fast path)
	val := t.Get(key)
	if !val.IsNil() {
		return val, nil
	}
	for depth := 0; depth < vm.MaxMetaDepth(); depth++ {
		// Key not found, check for __index metamethod
		mt := t.Metatable()
		if mt == nil {
			return Nil, nil
		}

		index := mt.Get(metaIndex)
		if index.IsNil() {
			return Nil, nil
		}

		if index.IsTable() {
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
		return vm.indexValue(index, key)
	}
	return Nil, vm.runtimeError("'__index' chain too long; possible loop")
}

// tableGetString gets a value from a table by string key, handling __index metamethod
func (vm *VM) tableGetString(t LuaTable, key string) (Value, error) {
	// Fast path: concrete table with no metatable
	if ct, ok := t.(*Table); ok && ct.metatable == nil {
		return ct.GetString(key), nil
	}
	// Initial table check (free, like Lua 5.4's inline bytecode fast path)
	if ct, ok := t.(*Table); ok {
		if val := ct.GetString(key); !val.IsNil() {
			return val, nil
		}
	} else {
		if val := t.Get(NewString(key)); !val.IsNil() {
			return val, nil
		}
	}
	for depth := 0; depth < vm.MaxMetaDepth(); depth++ {
		// Key not found, check for __index metamethod
		mt := t.Metatable()
		if mt == nil {
			return Nil, nil
		}

		index := mt.Get(metaIndex)
		if index.IsNil() {
			return Nil, nil
		}

		if index.IsTable() {
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
		return vm.indexValue(index, NewString(key))
	}
	return Nil, vm.runtimeError("'__index' chain too long; possible loop")
}

// tableGetInt gets a value from a table by int key, handling __index metamethod
func (vm *VM) tableGetInt(t LuaTable, key int) (Value, error) {
	// Fast path: concrete table with no metatable
	if ct, ok := t.(*Table); ok && ct.metatable == nil {
		return ct.GetInt(key), nil
	}
	// Initial table check (free, like Lua 5.4's inline bytecode fast path)
	if ct, ok := t.(*Table); ok {
		if val := ct.GetInt(key); !val.IsNil() {
			return val, nil
		}
	} else {
		if val := t.Get(NewInt(int64(key))); !val.IsNil() {
			return val, nil
		}
	}
	for depth := 0; depth < vm.MaxMetaDepth(); depth++ {
		// Key not found, check for __index metamethod
		mt := t.Metatable()
		if mt == nil {
			return Nil, nil
		}

		index := mt.Get(metaIndex)
		if index.IsNil() {
			return Nil, nil
		}

		if index.IsTable() {
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
		return vm.indexValue(index, NewInt(int64(key)))
	}
	return Nil, vm.runtimeError("'__index' chain too long; possible loop")
}

// indexValue tries to index a non-table value by looking up its metatable.
// For example, if __index is a string, this chains through the string metatable.
func (vm *VM) indexValue(val Value, key Value) (Value, error) {
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
		return Nil, vm.runtimeError("attempt to index a %s value", val.Type())
	}
	index := mt.Get(metaIndex)
	if index.IsNil() {
		return Nil, vm.runtimeError("attempt to index a %s value", val.Type())
	}
	if index.IsTable() {
		return vm.tableGet(index.AsTable(), key)
	}
	if index.IsFunction() || index.IsNativeFunc() {
		return vm.callMetamethod("index", index, val, key)
	}
	return Nil, vm.runtimeError("attempt to index a %s value", val.Type())
}

// resolveIndex resolves an __index metamethod for a non-table value.
// mm is the __index metamethod value, obj is the original value, key is the lookup key.
func (vm *VM) resolveIndex(mm Value, obj Value, key Value) (Value, error) {
	if mm.IsTable() {
		return vm.tableGet(mm.AsTable(), key)
	}
	if mm.IsFunction() || mm.IsNativeFunc() {
		return vm.callMetamethod("index", mm, obj, key)
	}
	return Nil, vm.runtimeError("attempt to index a %s value", obj.Type())
}

// tableSet sets a value in a table, handling __newindex metamethod.
// The loop structure matches Lua 5.4's luaV_finishset: the initial table check
// is "free" (not counted), and each loop iteration performs one redirect + check.
func (vm *VM) tableSet(t LuaTable, key, value Value) error {
	// Fast path: concrete table with no metatable (skip existence check)
	if ct, ok := t.(*Table); ok && ct.metatable == nil {
		if err := ct.Set(key, value); err != nil {
			return vm.runtimeError("%s", err)
		}
		return nil
	}
	// Initial check (free, like Lua 5.4's inline bytecode fast path)
	existing := t.Get(key)
	if !existing.IsNil() {
		if err := t.Set(key, value); err != nil {
			return vm.runtimeError("%s", err)
		}
		return nil
	}
	for depth := 0; depth < vm.MaxMetaDepth(); depth++ {
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

		// __newindex is a non-table, non-function value — Lua 5.4 chains
		// into it, which errors because it can't be indexed.
		return vm.runtimeError("attempt to index a %s value", vm.ObjTypeName(newindex))
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
	// Initial check (free, like Lua 5.4's inline bytecode fast path)
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
	for depth := 0; depth < vm.MaxMetaDepth(); depth++ {
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

			return vm.runtimeError("attempt to index a %s value", vm.ObjTypeName(newindex))
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

		return vm.runtimeError("attempt to index a %s value", vm.ObjTypeName(newindex))
	}
	return vm.runtimeError("'__newindex' chain too long; possible loop")
}

// tableSetInt sets a value in a table by int key, handling __newindex metamethod
func (vm *VM) tableSetInt(t LuaTable, key int, value Value) error {
	// Fast path: concrete table with no metatable
	if ct, ok := t.(*Table); ok && ct.metatable == nil {
		ct.SetInt(key, value)
		return nil
	}
	// Initial check (free, like Lua 5.4's inline bytecode fast path)
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
	for depth := 0; depth < vm.MaxMetaDepth(); depth++ {
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

			return vm.runtimeError("attempt to index a %s value", vm.ObjTypeName(newindex))
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

		return vm.runtimeError("attempt to index a %s value", vm.ObjTypeName(newindex))
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
	if mm := vm.getMetafield(val, "__newindex"); !mm.IsNil() {
		if mm.IsFunction() || mm.IsNativeFunc() {
			_, err := vm.callMetamethod3("newindex", mm, val, key, value)
			return err
		}
		if mm.IsTable() {
			return vm.tableSet(mm.AsTable(), key, value)
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
	if mm := vm.getMetafield(val, "__newindex"); !mm.IsNil() {
		if mm.IsFunction() || mm.IsNativeFunc() {
			_, err := vm.callMetamethod3("newindex", mm, val, NewInt(int64(key)), value)
			return err
		}
		if mm.IsTable() {
			return vm.tableSet(mm.AsTable(), NewInt(int64(key)), value)
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
	if val.IsTable() {
		return vm.tableGet(val.AsTable(), key)
	}
	return vm.indexValue(val, key)
}

// ObjLen returns #v with __len metamethod support.
func (vm *VM) ObjLen(val Value) (int, error) {
	if val.IsString() {
		return len(val.AsString()), nil
	}
	mm := vm.getMetafield(val, "__len")
	if !mm.IsNil() {
		res, err := vm.callMetamethod("len", mm, val, Nil)
		if err != nil {
			return 0, err
		}
		if i, ok := res.ToInt(); ok {
			return int(i), nil
		}
		return 0, fmt.Errorf("object length is not an integer")
	}
	if val.IsTable() {
		return val.AsTable().Len(), nil
	}
	return 0, vm.runtimeError("attempt to get length of a %s value", val.Type())
}
