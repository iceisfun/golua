package vm

import "fmt"

// Userdata represents a full userdata value in Lua. It holds an arbitrary
// Go value and an optional metatable. Methods on userdata are dispatched
// via the metatable's __index field.
//
// Lua 5.4 Reference: §2.1 (userdata).
type Userdata struct {
	Data      interface{} // Arbitrary Go value
	metatable LuaTable    // Optional metatable
	uservalue Value       // Single Lua 5.4 user value slot
}

// NewUserdataValue creates a Value of type userdata wrapping arbitrary Go data.
// The metatable controls method dispatch and metamethod behavior.
func NewUserdataValue(data interface{}, mt LuaTable) Value {
	ud := &Userdata{Data: data, metatable: mt, uservalue: Nil}
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

// UserValue returns the userdata's single user value slot.
func (u *Userdata) UserValue() Value {
	return u.uservalue
}

// SetUserValue replaces the userdata's single user value slot.
func (u *Userdata) SetUserValue(v Value) {
	u.uservalue = v
}

// String returns a string representation of the userdata.
func (u *Userdata) String() string {
	return fmt.Sprintf("userdata: %p", u)
}
