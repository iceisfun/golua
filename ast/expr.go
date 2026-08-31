package ast

import "github.com/iceisfun/golua/v2/token"

// ---------------------------------------------------------------------------
// Expressions
// ---------------------------------------------------------------------------

// posAfter returns the position n bytes past p on the same line. It is used to
// compute the End() of single-line leaf tokens (keywords, names, numbers)
// whose length is known but whose closing position is not separately recorded.
func posAfter(p token.Pos, n int) token.Pos {
	p.Offset += n
	p.Column += n
	return p
}

// endOr returns end if it carries a real position, else the fallback. Nodes
// constructed directly (e.g. in tests) without an explicit end position fall
// back to a sensible computed value instead of a zero Pos.
// endOrLazy is endOr for nodes whose fallback is the end of a child. The child
// is only consulted when this node has no end position of its own: these nodes
// nest down their left spine, so evaluating the fallback eagerly walks the
// whole chain on every call and a long chain exhausts the goroutine stack.
func endOrLazy(end token.Pos, fallback Expr) token.Pos {
	if end.Line != 0 {
		return end
	}
	if fallback == nil {
		return end
	}
	return fallback.End()
}

func endOr(end, fallback token.Pos) token.Pos {
	if end.Line != 0 {
		return end
	}
	return fallback
}

// NilExpr represents the literal `nil`.
type NilExpr struct{ P token.Pos }

func (e *NilExpr) Pos() token.Pos { return e.P }
func (e *NilExpr) End() token.Pos { return posAfter(e.P, 3) }
func (*NilExpr) exprTag()         {}

func NewNilExpr(p token.Pos) *NilExpr { return &NilExpr{P: p} }

// TrueExpr represents `true`.
type TrueExpr struct{ P token.Pos }

func (e *TrueExpr) Pos() token.Pos { return e.P }
func (e *TrueExpr) End() token.Pos { return posAfter(e.P, 4) }
func (*TrueExpr) exprTag()         {}

func NewTrueExpr(p token.Pos) *TrueExpr { return &TrueExpr{P: p} }

// FalseExpr represents `false`.
type FalseExpr struct{ P token.Pos }

func (e *FalseExpr) Pos() token.Pos { return e.P }
func (e *FalseExpr) End() token.Pos { return posAfter(e.P, 5) }
func (*FalseExpr) exprTag()         {}

func NewFalseExpr(p token.Pos) *FalseExpr { return &FalseExpr{P: p} }

// NumberExpr represents an integer literal.
type NumberExpr struct {
	P     token.Pos
	Value int64
	Raw   string
}

func (e *NumberExpr) Pos() token.Pos { return e.P }
func (e *NumberExpr) End() token.Pos { return posAfter(e.P, len(e.Raw)) }
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
func (e *FloatExpr) End() token.Pos { return posAfter(e.P, len(e.Raw)) }
func (*FloatExpr) exprTag()         {}

func NewFloatExpr(p token.Pos, v float64, raw string) *FloatExpr {
	return &FloatExpr{P: p, Value: v, Raw: raw}
}

// StringExpr represents a string literal. EndP records the position just past
// the closing delimiter (string literals may span lines via long brackets).
type StringExpr struct {
	P     token.Pos
	EndP  token.Pos
	Value string
}

func (e *StringExpr) Pos() token.Pos { return e.P }
func (e *StringExpr) End() token.Pos { return endOr(e.EndP, posAfter(e.P, len(e.Value))) }
func (*StringExpr) exprTag()         {}

func NewStringExpr(p token.Pos, v string) *StringExpr {
	return &StringExpr{P: p, Value: v}
}

// VarArgExpr represents `...`.
type VarArgExpr struct{ P token.Pos }

func (e *VarArgExpr) Pos() token.Pos { return e.P }
func (e *VarArgExpr) End() token.Pos { return posAfter(e.P, 3) }
func (*VarArgExpr) exprTag()         {}

func NewVarArgExpr(p token.Pos) *VarArgExpr { return &VarArgExpr{P: p} }

// NameExpr represents a simple name reference.
type NameExpr struct {
	P    token.Pos
	Name string
}

func (e *NameExpr) Pos() token.Pos { return e.P }
func (e *NameExpr) End() token.Pos { return posAfter(e.P, len(e.Name)) }
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
func (e *BinopExpr) End() token.Pos { return e.Right.End() }
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
func (e *UnopExpr) End() token.Pos { return e.Operand.End() }
func (*UnopExpr) exprTag()         {}

func NewUnopExpr(p token.Pos, op string, operand Expr) *UnopExpr {
	return &UnopExpr{P: p, Op: op, Operand: operand}
}

// IndexExpr represents table[key]. EndP records the closing ']'.
type IndexExpr struct {
	P     token.Pos
	EndP  token.Pos
	Table Expr
	Key   Expr
}

func (e *IndexExpr) Pos() token.Pos { return e.P }
func (e *IndexExpr) End() token.Pos { return endOrLazy(e.EndP, e.Key) }
func (*IndexExpr) exprTag()         {}

func NewIndexExpr(p token.Pos, table, key Expr) *IndexExpr {
	return &IndexExpr{P: p, Table: table, Key: key}
}

// FieldExpr represents table.field. EndP records the end of the field name.
type FieldExpr struct {
	P     token.Pos
	EndP  token.Pos
	Table Expr
	Field string
}

func (e *FieldExpr) Pos() token.Pos { return e.P }
func (e *FieldExpr) End() token.Pos { return endOrLazy(e.EndP, e.Table) }
func (*FieldExpr) exprTag()         {}

func NewFieldExpr(p token.Pos, table Expr, field string) *FieldExpr {
	return &FieldExpr{P: p, Table: table, Field: field}
}

// MethodCallExpr represents table:method(args). EndP records the closing
// delimiter of the call (')' or the end of a string/table argument).
type MethodCallExpr struct {
	P      token.Pos
	EndP   token.Pos
	Object Expr
	Method string
	Args   []Expr
}

func (e *MethodCallExpr) Pos() token.Pos { return e.P }
func (e *MethodCallExpr) End() token.Pos { return endOrLazy(e.EndP, e.Object) }
func (*MethodCallExpr) exprTag()         {}

func NewMethodCallExpr(p token.Pos, obj Expr, method string, args []Expr) *MethodCallExpr {
	return &MethodCallExpr{P: p, Object: obj, Method: method, Args: args}
}

// FuncCallExpr represents func(args). EndP records the closing delimiter of
// the call (')' or the end of a string/table argument).
type FuncCallExpr struct {
	P    token.Pos
	EndP token.Pos
	Func Expr
	Args []Expr
}

func (e *FuncCallExpr) Pos() token.Pos { return e.P }
func (e *FuncCallExpr) End() token.Pos { return endOrLazy(e.EndP, e.Func) }
func (*FuncCallExpr) exprTag()         {}

func NewFuncCallExpr(p token.Pos, fn Expr, args []Expr) *FuncCallExpr {
	return &FuncCallExpr{P: p, Func: fn, Args: args}
}

// FuncExpr represents an anonymous function: function(params) body end.
// EndP records the closing 'end' keyword; EndLine is retained for callers that
// only need the line.
type FuncExpr struct {
	P          token.Pos
	EndP       token.Pos
	Params     []*NameExpr
	VarArg     bool
	VarArgName string // Lua 5.5: `... name` captures varargs
	Body       *Block
	EndLine    int // line of the closing 'end' keyword (0 if unknown)
}

func (e *FuncExpr) Pos() token.Pos { return e.P }
func (e *FuncExpr) End() token.Pos { return endOr(e.EndP, posAfter(e.P, 8)) }
func (*FuncExpr) exprTag()         {}

func NewFuncExpr(p token.Pos, params []*NameExpr, vararg bool, vaName string, body *Block) *FuncExpr {
	return &FuncExpr{P: p, Params: params, VarArg: vararg, VarArgName: vaName, Body: body}
}

// TableConstructor represents { field, field, ... }. EndP records the
// closing '}'.
type TableConstructor struct {
	P      token.Pos
	EndP   token.Pos
	Fields []*TableField
}

func (e *TableConstructor) Pos() token.Pos { return e.P }
func (e *TableConstructor) End() token.Pos { return endOr(e.EndP, posAfter(e.P, 1)) }
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

func (e *TableField) End() token.Pos {
	if e.Value != nil {
		return e.Value.End()
	}
	if e.Key != nil {
		return e.Key.End()
	}
	return e.P
}

func NewTableField(p token.Pos, key, value Expr) *TableField {
	return &TableField{P: p, Key: key, Value: value}
}

// ParenExpr represents (expr). EndP records the closing ')'.
type ParenExpr struct {
	P     token.Pos
	EndP  token.Pos
	Inner Expr
}

func (e *ParenExpr) Pos() token.Pos { return e.P }
func (e *ParenExpr) End() token.Pos { return endOrLazy(e.EndP, e.Inner) }
func (*ParenExpr) exprTag()         {}

func NewParenExpr(p token.Pos, inner Expr) *ParenExpr {
	return &ParenExpr{P: p, Inner: inner}
}
