package vm

import "fmt"

// tableGet gets a value from a table, handling __index metamethod
func (vm *VM) tableGet(t LuaTable, key Value) (Value, error) {
	// Fast path: concrete table with no metatable (most common case)
	if ct, ok := t.(*Table); ok && ct.metatable == nil {
		return ct.Get(key), nil
	}
	for depth := 0; depth <= vm.MaxMetaDepth(); depth++ {
		val := t.Get(key)
		if !val.IsNil() {
			return val, nil
		}

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
			continue
		}

		if index.IsFunction() || index.IsNativeFunc() {
			// __index is a function, call it with (table, key)
			return vm.callMetamethod("index", index, NewTable(t), key)
		}

		// __index is another value — chain through its metatable
		return vm.indexValue(index, key)
	}
	return Nil, fmt.Errorf("'__index' chain too long; possible loop")
}

// tableGetString gets a value from a table by string key, handling __index metamethod
func (vm *VM) tableGetString(t LuaTable, key string) (Value, error) {
	// Fast path: concrete table with no metatable
	if ct, ok := t.(*Table); ok && ct.metatable == nil {
		return ct.GetString(key), nil
	}
	for depth := 0; depth <= vm.MaxMetaDepth(); depth++ {
		// Fast path: use *Table.GetString to avoid NewString allocation
		var val Value
		if ct, ok := t.(*Table); ok {
			val = ct.GetString(key)
		} else {
			val = t.Get(NewString(key))
		}
		if !val.IsNil() {
			return val, nil
		}

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
			continue
		}

		if index.IsFunction() || index.IsNativeFunc() {
			// __index is a function, call it with (table, key)
			return vm.callMetamethod("index", index, NewTable(t), NewString(key))
		}

		// __index is another value — chain through its metatable
		return vm.indexValue(index, NewString(key))
	}
	return Nil, fmt.Errorf("'__index' chain too long; possible loop")
}

// tableGetInt gets a value from a table by int key, handling __index metamethod
func (vm *VM) tableGetInt(t LuaTable, key int) (Value, error) {
	// Fast path: concrete table with no metatable
	if ct, ok := t.(*Table); ok && ct.metatable == nil {
		return ct.GetInt(key), nil
	}
	for depth := 0; depth <= vm.MaxMetaDepth(); depth++ {
		// Fast path: use *Table.GetInt to avoid NewInt/hashKey overhead
		var val Value
		if ct, ok := t.(*Table); ok {
			val = ct.GetInt(key)
		} else {
			val = t.Get(NewInt(int64(key)))
		}
		if !val.IsNil() {
			return val, nil
		}

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
			continue
		}

		if index.IsFunction() || index.IsNativeFunc() {
			// __index is a function, call it with (table, key)
			return vm.callMetamethod("index", index, NewTable(t), NewInt(int64(key)))
		}

		// __index is another value — chain through its metatable
		return vm.indexValue(index, NewInt(int64(key)))
	}
	return Nil, fmt.Errorf("'__index' chain too long; possible loop")
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

// tableSet sets a value in a table, handling __newindex metamethod
func (vm *VM) tableSet(t LuaTable, key, value Value) error {
	// Fast path: concrete table with no metatable (skip existence check)
	if ct, ok := t.(*Table); ok && ct.metatable == nil {
		return ct.Set(key, value)
	}
	for depth := 0; depth <= vm.MaxMetaDepth(); depth++ {
		// Check if key already exists (raw access)
		existing := t.Get(key)
		if !existing.IsNil() {
			// Key exists, set directly
			if err := t.Set(key, value); err != nil {
				return err
			}
			return nil
		}

		// Key doesn't exist, check for __newindex metamethod
		mt := t.Metatable()
		if mt == nil {
			if err := t.Set(key, value); err != nil {
				return err
			}
			return nil
		}

		newindex := mt.Get(metaNewIndex)
		if newindex.IsNil() {
			if err := t.Set(key, value); err != nil {
				return err
			}
			return nil
		}

		if newindex.IsTable() {
			// __newindex is a table, follow the chain
			t = newindex.AsTable()
			continue
		}

		if newindex.IsFunction() || newindex.IsNativeFunc() {
			// __newindex is a function, call it with (table, key, value)
			_, err := vm.callMetamethod3("newindex", newindex, NewTable(t), key, value)
			return err
		}

		// __newindex is a non-table, non-function value — error
		return fmt.Errorf("'__newindex' is not a table or function")
	}
	return fmt.Errorf("'__newindex' chain too long; possible loop")
}

// tableSetString sets a value in a table by string key, handling __newindex metamethod
func (vm *VM) tableSetString(t LuaTable, key string, value Value) error {
	// Fast path: concrete table with no metatable
	if ct, ok := t.(*Table); ok && ct.metatable == nil {
		ct.SetString(key, value)
		return nil
	}
	for depth := 0; depth <= vm.MaxMetaDepth(); depth++ {
		// Fast path: use *Table methods to avoid NewString allocation
		if ct, ok := t.(*Table); ok {
			existing := ct.GetString(key)
			if !existing.IsNil() {
				ct.SetString(key, value)
				return nil
			}

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
				continue
			}

			if newindex.IsFunction() || newindex.IsNativeFunc() {
				_, err := vm.callMetamethod3("newindex", newindex, NewTable(t), NewString(key), value)
				return err
			}

			return fmt.Errorf("'__newindex' is not a table or function")
		}

		// Slow path: generic LuaTable interface
		keyVal := NewString(key)
		existing := t.Get(keyVal)
		if !existing.IsNil() {
			if err := t.Set(keyVal, value); err != nil {
				return err
			}
			return nil
		}

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
			continue
		}

		if newindex.IsFunction() || newindex.IsNativeFunc() {
			_, err := vm.callMetamethod3("newindex", newindex, NewTable(t), keyVal, value)
			return err
		}

		return fmt.Errorf("'__newindex' is not a table or function")
	}
	return fmt.Errorf("'__newindex' chain too long; possible loop")
}

// tableSetInt sets a value in a table by int key, handling __newindex metamethod
func (vm *VM) tableSetInt(t LuaTable, key int, value Value) error {
	// Fast path: concrete table with no metatable
	if ct, ok := t.(*Table); ok && ct.metatable == nil {
		ct.SetInt(key, value)
		return nil
	}
	for depth := 0; depth <= vm.MaxMetaDepth(); depth++ {
		// Fast path: use *Table methods to avoid NewInt/hashKey overhead
		if ct, ok := t.(*Table); ok {
			existing := ct.GetInt(key)
			if !existing.IsNil() {
				ct.SetInt(key, value)
				return nil
			}

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
				continue
			}

			if newindex.IsFunction() || newindex.IsNativeFunc() {
				_, err := vm.callMetamethod3("newindex", newindex, NewTable(t), NewInt(int64(key)), value)
				return err
			}

			return fmt.Errorf("'__newindex' is not a table or function")
		}

		// Slow path: generic LuaTable interface
		keyVal := NewInt(int64(key))
		existing := t.Get(keyVal)
		if !existing.IsNil() {
			if err := t.Set(keyVal, value); err != nil {
				return err
			}
			return nil
		}

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
			continue
		}

		if newindex.IsFunction() || newindex.IsNativeFunc() {
			_, err := vm.callMetamethod3("newindex", newindex, NewTable(t), keyVal, value)
			return err
		}

		return fmt.Errorf("'__newindex' is not a table or function")
	}
	return fmt.Errorf("'__newindex' chain too long; possible loop")
}

// TableGet retrieves t[key] with __index metamethod support.
func (vm *VM) TableGet(t LuaTable, key Value) (Value, error) {
	return vm.tableGet(t, key)
}

// TableGetInt retrieves t[key] with __index metamethod support.
func (vm *VM) TableGetInt(t LuaTable, key int) (Value, error) {
	return vm.tableGetInt(t, key)
}

// TableSetInt sets t[key]=value with __newindex metamethod support.
func (vm *VM) TableSetInt(t LuaTable, key int, value Value) error {
	return vm.tableSetInt(t, key, value)
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
