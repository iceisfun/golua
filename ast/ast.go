// Package ast defines the abstract syntax tree for Lua 5.5.
package ast

import "github.com/iceisfun/golua/token"

// ---------------------------------------------------------------------------
// Interfaces
// ---------------------------------------------------------------------------

// Node is the common interface for every AST node.
type Node interface {
	Pos() token.Pos
}

// Stmt is a statement node.
type Stmt interface {
	Node
	stmtTag()
}

// Expr is an expression node.
type Expr interface {
	Node
	exprTag()
}

// ---------------------------------------------------------------------------
// Program (top-level)
// ---------------------------------------------------------------------------

// Block is a list of statements.
type Block struct {
	Start token.Pos
	Stmts []Stmt
}

func (b *Block) Pos() token.Pos { return b.Start }

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

// ---------------------------------------------------------------------------
// Statements
// ---------------------------------------------------------------------------

// AssignStmt: targets = values
type AssignStmt struct {
	P       token.Pos
	Targets []Expr
	Values  []Expr
}

func (s *AssignStmt) Pos() token.Pos { return s.P }
func (*AssignStmt) stmtTag()         {}

func NewAssignStmt(p token.Pos, targets, values []Expr) *AssignStmt {
	return &AssignStmt{P: p, Targets: targets, Values: values}
}

// LocalStmt: local name, name = expr, expr
type LocalStmt struct {
	P       token.Pos
	Names   []*NameExpr
	Attribs []string // "", "const", "close"
	Values  []Expr
}

func (s *LocalStmt) Pos() token.Pos { return s.P }
func (*LocalStmt) stmtTag()         {}

func NewLocalStmt(p token.Pos, names []*NameExpr, attribs []string, values []Expr) *LocalStmt {
	return &LocalStmt{P: p, Names: names, Attribs: attribs, Values: values}
}

// DoStmt: do block end
type DoStmt struct {
	P    token.Pos
	Body *Block
}

func (s *DoStmt) Pos() token.Pos { return s.P }
func (*DoStmt) stmtTag()         {}

func NewDoStmt(p token.Pos, body *Block) *DoStmt {
	return &DoStmt{P: p, Body: body}
}

// WhileStmt: while cond do block end
type WhileStmt struct {
	P    token.Pos
	Cond Expr
	Body *Block
}

func (s *WhileStmt) Pos() token.Pos { return s.P }
func (*WhileStmt) stmtTag()         {}

func NewWhileStmt(p token.Pos, cond Expr, body *Block) *WhileStmt {
	return &WhileStmt{P: p, Cond: cond, Body: body}
}

// RepeatStmt: repeat block until cond
type RepeatStmt struct {
	P    token.Pos
	Body *Block
	Cond Expr
}

func (s *RepeatStmt) Pos() token.Pos { return s.P }
func (*RepeatStmt) stmtTag()         {}

func NewRepeatStmt(p token.Pos, body *Block, cond Expr) *RepeatStmt {
	return &RepeatStmt{P: p, Body: body, Cond: cond}
}

// IfStmt: if cond then block {elseif} [else] end
type IfStmt struct {
	P       token.Pos
	Cond    Expr
	Then    *Block
	ElseIfs []*ElseIf
	Else    *Block // nil if no else
}

func (s *IfStmt) Pos() token.Pos { return s.P }
func (*IfStmt) stmtTag()         {}

func NewIfStmt(p token.Pos, cond Expr, then *Block, elseifs []*ElseIf, els *Block) *IfStmt {
	return &IfStmt{P: p, Cond: cond, Then: then, ElseIfs: elseifs, Else: els}
}

// ElseIf is an elseif branch.
type ElseIf struct {
	P    token.Pos
	Cond Expr
	Then *Block
}

func (e *ElseIf) Pos() token.Pos { return e.P }

func NewElseIf(p token.Pos, cond Expr, then *Block) *ElseIf {
	return &ElseIf{P: p, Cond: cond, Then: then}
}

// ForNumStmt: for name = start, stop [, step] do block end
type ForNumStmt struct {
	P     token.Pos
	Name  *NameExpr
	Start Expr
	Stop  Expr
	Step  Expr // nil = default 1
	Body  *Block
}

func (s *ForNumStmt) Pos() token.Pos { return s.P }
func (*ForNumStmt) stmtTag()         {}

func NewForNumStmt(p token.Pos, name *NameExpr, start, stop, step Expr, body *Block) *ForNumStmt {
	return &ForNumStmt{P: p, Name: name, Start: start, Stop: stop, Step: step, Body: body}
}

// ForInStmt: for names in exprs do block end
type ForInStmt struct {
	P     token.Pos
	Names []*NameExpr
	Iters []Expr
	Body  *Block
}

func (s *ForInStmt) Pos() token.Pos { return s.P }
func (*ForInStmt) stmtTag()         {}

func NewForInStmt(p token.Pos, names []*NameExpr, iters []Expr, body *Block) *ForInStmt {
	return &ForInStmt{P: p, Names: names, Iters: iters, Body: body}
}

// ReturnStmt: return [exprs] [;]
type ReturnStmt struct {
	P      token.Pos
	Values []Expr
}

func (s *ReturnStmt) Pos() token.Pos { return s.P }
func (*ReturnStmt) stmtTag()         {}

func NewReturnStmt(p token.Pos, values []Expr) *ReturnStmt {
	return &ReturnStmt{P: p, Values: values}
}

// BreakStmt: break
type BreakStmt struct{ P token.Pos }

func (s *BreakStmt) Pos() token.Pos { return s.P }
func (*BreakStmt) stmtTag()         {}

func NewBreakStmt(p token.Pos) *BreakStmt { return &BreakStmt{P: p} }

// GotoStmt: goto label
type GotoStmt struct {
	P     token.Pos
	Label string
}

func (s *GotoStmt) Pos() token.Pos { return s.P }
func (*GotoStmt) stmtTag()         {}

func NewGotoStmt(p token.Pos, label string) *GotoStmt {
	return &GotoStmt{P: p, Label: label}
}

// LabelStmt: ::label::
type LabelStmt struct {
	P    token.Pos
	Name string
}

func (s *LabelStmt) Pos() token.Pos { return s.P }
func (*LabelStmt) stmtTag()         {}

func NewLabelStmt(p token.Pos, name string) *LabelStmt {
	return &LabelStmt{P: p, Name: name}
}

// ExprStmt wraps a function call used as a statement.
type ExprStmt struct {
	P    token.Pos
	Expr Expr
}

func (s *ExprStmt) Pos() token.Pos { return s.P }
func (*ExprStmt) stmtTag()         {}

func NewExprStmt(p token.Pos, expr Expr) *ExprStmt {
	return &ExprStmt{P: p, Expr: expr}
}

// FuncStmt: function name.field:method(params) body end
type FuncStmt struct {
	P        token.Pos
	Name     Expr // NameExpr or FieldExpr chain
	IsMethod bool
	Func     *FuncExpr
}

func (s *FuncStmt) Pos() token.Pos { return s.P }
func (*FuncStmt) stmtTag()         {}

func NewFuncStmt(p token.Pos, name Expr, isMethod bool, fn *FuncExpr) *FuncStmt {
	return &FuncStmt{P: p, Name: name, IsMethod: isMethod, Func: fn}
}

// LocalFuncStmt: local function name(params) body end
type LocalFuncStmt struct {
	P    token.Pos
	Name *NameExpr
	Func *FuncExpr
}

func (s *LocalFuncStmt) Pos() token.Pos { return s.P }
func (*LocalFuncStmt) stmtTag()         {}

func NewLocalFuncStmt(p token.Pos, name *NameExpr, fn *FuncExpr) *LocalFuncStmt {
	return &LocalFuncStmt{P: p, Name: name, Func: fn}
}

// GlobalStmt: global [attrib] name, name = expr, expr (Lua 5.5)
type GlobalStmt struct {
	P       token.Pos
	Names   []*NameExpr
	Attribs []string
	Values  []Expr
	Star    bool // global *
}

func (s *GlobalStmt) Pos() token.Pos { return s.P }
func (*GlobalStmt) stmtTag()         {}

func NewGlobalStmt(p token.Pos, names []*NameExpr, attribs []string, values []Expr) *GlobalStmt {
	return &GlobalStmt{P: p, Names: names, Attribs: attribs, Values: values}
}

func NewGlobalStarStmt(p token.Pos, attrib string) *GlobalStmt {
	return &GlobalStmt{P: p, Star: true, Attribs: []string{attrib}}
}

// GlobalFuncStmt: global function name(params) body end (Lua 5.5)
type GlobalFuncStmt struct {
	P    token.Pos
	Name *NameExpr
	Func *FuncExpr
}

func (s *GlobalFuncStmt) Pos() token.Pos { return s.P }
func (*GlobalFuncStmt) stmtTag()         {}

func NewGlobalFuncStmt(p token.Pos, name *NameExpr, fn *FuncExpr) *GlobalFuncStmt {
	return &GlobalFuncStmt{P: p, Name: name, Func: fn}
}

// EmptyStmt: ;
type EmptyStmt struct{ P token.Pos }

func (s *EmptyStmt) Pos() token.Pos { return s.P }
func (*EmptyStmt) stmtTag()         {}

func NewEmptyStmt(p token.Pos) *EmptyStmt { return &EmptyStmt{P: p} }
