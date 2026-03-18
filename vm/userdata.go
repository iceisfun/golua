package vm

import "fmt"

// Userdata represents a full userdata value in Lua. It holds an arbitrary
// Go value and an optional metatable. Methods on userdata are dispatched
// via the metatable's __index field.
//
// Lua 5.4 Reference: §2.1 (userdata).
type Userdata struct {
	Data       interface{} // Arbitrary Go value
	metatable  LuaTable    // Optional metatable
	uservalues []Value     // User value slots (0-255)
}

// NewUserdataValue creates a Value of type userdata wrapping arbitrary Go data
// with 1 user value slot (the default for lua_newuserdata).
// The metatable controls method dispatch and metamethod behavior.
func NewUserdataValue(data interface{}, mt LuaTable) Value {
	ud := &Userdata{Data: data, metatable: mt, uservalues: make([]Value, 1)}
	ud.uservalues[0] = Nil
	return Value{typ: typeUpvalue, ptr: ud}
}

// NewUserdataValueUV creates a Value of type userdata wrapping arbitrary Go data
// with the specified number of user value slots (matching lua_newuserdatauv).
func NewUserdataValueUV(data interface{}, mt LuaTable, nuvalue int) Value {
	var uv []Value
	if nuvalue > 0 {
		uv = make([]Value, nuvalue)
		for i := range uv {
			uv[i] = Nil
		}
	}
	ud := &Userdata{Data: data, metatable: mt, uservalues: uv}
	return Value{typ: typeUpvalue, ptr: ud}
}

// IsUserdata reports whether v is a full userdata value (as opposed to a
// lightuserdata/upvalue-id).
func (v Value) IsUserdata() bool {
	if v.typ != typeUpvalue {
		return false
	}
	_, ok := v.ptr.(*Userdata)
	return ok
}

// IsLightUserdata reports whether v is a light userdata value (upvalue ID),
// as opposed to a full userdata.
func (v Value) IsLightUserdata() bool {
	if v.typ != typeUpvalue {
		return false
	}
	_, ok := v.ptr.(*Userdata)
	return !ok
}

// AsUserdata returns the Userdata struct if v is a full userdata, or nil.
func (v Value) AsUserdata() *Userdata {
	if v.typ != typeUpvalue {
		return nil
	}
	ud, _ := v.ptr.(*Userdata)
	return ud
}

// Metatable returns the userdata's metatable, or nil.
func (u *Userdata) Metatable() LuaTable {
	return u.metatable
}

// SetMetatable sets the userdata's metatable.
func (u *Userdata) SetMetatable(mt LuaTable) {
	u.metatable = mt
}

// UserValueCount returns the number of user value slots.
func (u *Userdata) UserValueCount() int {
	return len(u.uservalues)
}

// GetUserValue returns the user value at slot n (1-based).
// Returns the value and true if n is in range, or Nil and false otherwise.
func (u *Userdata) GetUserValue(n int) (Value, bool) {
	if n < 1 || n > len(u.uservalues) {
		return Nil, false
	}
	return u.uservalues[n-1], true
}

// SetUserValue sets the user value at slot n (1-based).
// Returns true if n is in range, false otherwise.
func (u *Userdata) SetUserValue(n int, val Value) bool {
	if n < 1 || n > len(u.uservalues) {
		return false
	}
	u.uservalues[n-1] = val
	return true
}

// UserValue returns the first user value slot (for backward compatibility).
// Deprecated: Use GetUserValue(1) instead.
func (u *Userdata) UserValue() Value {
	v, _ := u.GetUserValue(1)
	return v
}

// String returns a string representation of the userdata.
func (u *Userdata) String() string {
	return fmt.Sprintf("userdata: %p", u)
}
