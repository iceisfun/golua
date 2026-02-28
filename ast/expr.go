package ast

import "github.com/iceisfun/golua/token"

// ---------------------------------------------------------------------------
// Expressions
// ---------------------------------------------------------------------------

// NilExpr represents the literal `nil`.
type NilExpr struct{ P token.Pos }

func (e *NilExpr) Pos() token.Pos { return e.P }
func (*NilExpr) exprTag()         {}

func NewNilExpr(p token.Pos) *NilExpr { return &NilExpr{P: p} }

// TrueExpr represents `true`.
type TrueExpr struct{ P token.Pos }

func (e *TrueExpr) Pos() token.Pos { return e.P }
func (*TrueExpr) exprTag()         {}

func NewTrueExpr(p token.Pos) *TrueExpr { return &TrueExpr{P: p} }

// FalseExpr represents `false`.
type FalseExpr struct{ P token.Pos }

func (e *FalseExpr) Pos() token.Pos { return e.P }
func (*FalseExpr) exprTag()         {}

func NewFalseExpr(p token.Pos) *FalseExpr { return &FalseExpr{P: p} }

// NumberExpr represents an integer literal.
type NumberExpr struct {
	P     token.Pos
	Value int64
	Raw   string
}

func (e *NumberExpr) Pos() token.Pos { return e.P }
func (*NumberExpr) exprTag()         {}

func NewNumberExpr(p token.Pos, v int64, raw string) *NumberExpr {
	return &NumberExpr{P: p, Value: v, Raw: raw}
}

// FloatExpr represents a floating-point literal.
type FloatExpr struct {
	P     token.Pos
	Value float64
	Raw   string
}

func (e *FloatExpr) Pos() token.Pos { return e.P }
func (*FloatExpr) exprTag()         {}

func NewFloatExpr(p token.Pos, v float64, raw string) *FloatExpr {
	return &FloatExpr{P: p, Value: v, Raw: raw}
}

// StringExpr represents a string literal.
type StringExpr struct {
	P     token.Pos
	Value string
}

func (e *StringExpr) Pos() token.Pos { return e.P }
func (*StringExpr) exprTag()         {}

func NewStringExpr(p token.Pos, v string) *StringExpr {
	return &StringExpr{P: p, Value: v}
}

// VarArgExpr represents `...`.
type VarArgExpr struct{ P token.Pos }

func (e *VarArgExpr) Pos() token.Pos { return e.P }
func (*VarArgExpr) exprTag()         {}

func NewVarArgExpr(p token.Pos) *VarArgExpr { return &VarArgExpr{P: p} }

// NameExpr represents a simple name reference.
type NameExpr struct {
	P    token.Pos
	Name string
}

func (e *NameExpr) Pos() token.Pos { return e.P }
func (*NameExpr) exprTag()         {}

func NewNameExpr(p token.Pos, name string) *NameExpr {
	return &NameExpr{P: p, Name: name}
}

// BinopExpr represents a binary operation: left op right.
type BinopExpr struct {
	P     token.Pos
	Op    string
	Left  Expr
	Right Expr
}

func (e *BinopExpr) Pos() token.Pos { return e.P }
func (*BinopExpr) exprTag()         {}

func NewBinopExpr(p token.Pos, op string, left, right Expr) *BinopExpr {
	return &BinopExpr{P: p, Op: op, Left: left, Right: right}
}

// UnopExpr represents a unary operation: op operand.
type UnopExpr struct {
	P       token.Pos
	Op      string
	Operand Expr
}

func (e *UnopExpr) Pos() token.Pos { return e.P }
func (*UnopExpr) exprTag()         {}

func NewUnopExpr(p token.Pos, op string, operand Expr) *UnopExpr {
	return &UnopExpr{P: p, Op: op, Operand: operand}
}

// IndexExpr represents table[key].
type IndexExpr struct {
	P     token.Pos
	Table Expr
	Key   Expr
}

func (e *IndexExpr) Pos() token.Pos { return e.P }
func (*IndexExpr) exprTag()         {}

func NewIndexExpr(p token.Pos, table, key Expr) *IndexExpr {
	return &IndexExpr{P: p, Table: table, Key: key}
}

// FieldExpr represents table.field.
type FieldExpr struct {
	P     token.Pos
	Table Expr
	Field string
}

func (e *FieldExpr) Pos() token.Pos { return e.P }
func (*FieldExpr) exprTag()         {}

func NewFieldExpr(p token.Pos, table Expr, field string) *FieldExpr {
	return &FieldExpr{P: p, Table: table, Field: field}
}

// MethodCallExpr represents table:method(args).
type MethodCallExpr struct {
	P      token.Pos
	Object Expr
	Method string
	Args   []Expr
}

func (e *MethodCallExpr) Pos() token.Pos { return e.P }
func (*MethodCallExpr) exprTag()         {}

func NewMethodCallExpr(p token.Pos, obj Expr, method string, args []Expr) *MethodCallExpr {
	return &MethodCallExpr{P: p, Object: obj, Method: method, Args: args}
}

// FuncCallExpr represents func(args).
type FuncCallExpr struct {
	P    token.Pos
	Func Expr
	Args []Expr
}

func (e *FuncCallExpr) Pos() token.Pos { return e.P }
func (*FuncCallExpr) exprTag()         {}

func NewFuncCallExpr(p token.Pos, fn Expr, args []Expr) *FuncCallExpr {
	return &FuncCallExpr{P: p, Func: fn, Args: args}
}

// FuncExpr represents an anonymous function: function(params) body end.
type FuncExpr struct {
	P          token.Pos
	Params     []*NameExpr
	VarArg     bool
	VarArgName string // Lua 5.5: `... name` captures varargs
	Body       *Block
}

func (e *FuncExpr) Pos() token.Pos { return e.P }
func (*FuncExpr) exprTag()         {}

func NewFuncExpr(p token.Pos, params []*NameExpr, vararg bool, vaName string, body *Block) *FuncExpr {
	return &FuncExpr{P: p, Params: params, VarArg: vararg, VarArgName: vaName, Body: body}
}

// TableConstructor represents { field, field, ... }.
type TableConstructor struct {
	P      token.Pos
	Fields []*TableField
}

func (e *TableConstructor) Pos() token.Pos { return e.P }
func (*TableConstructor) exprTag()         {}

func NewTableConstructor(p token.Pos, fields []*TableField) *TableConstructor {
	return &TableConstructor{P: p, Fields: fields}
}

// TableField is one entry in a table constructor.
// Key is nil for list-style fields.
type TableField struct {
	P     token.Pos
	Key   Expr // nil for positional
	Value Expr
}

func (e *TableField) Pos() token.Pos { return e.P }

func NewTableField(p token.Pos, key, value Expr) *TableField {
	return &TableField{P: p, Key: key, Value: value}
}

// ParenExpr represents (expr).
type ParenExpr struct {
	P     token.Pos
	Inner Expr
}

func (e *ParenExpr) Pos() token.Pos { return e.P }
func (*ParenExpr) exprTag()         {}

func NewParenExpr(p token.Pos, inner Expr) *ParenExpr {
	return &ParenExpr{P: p, Inner: inner}
}
