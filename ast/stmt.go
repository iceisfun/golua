package ast

import "github.com/iceisfun/golua/token"

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
	P       token.Pos
	EndP    token.Pos // position of the closing 'end' keyword
	Body    *Block
	EndLine int // line of the 'end' keyword
}

func (s *DoStmt) Pos() token.Pos { return s.P }
func (*DoStmt) stmtTag()         {}

func NewDoStmt(p token.Pos, body *Block, endLine int) *DoStmt {
	return &DoStmt{P: p, Body: body, EndLine: endLine}
}

// WhileStmt: while cond do block end
type WhileStmt struct {
	P       token.Pos
	EndP    token.Pos // position of the closing 'end' keyword
	Cond    Expr
	Body    *Block
	EndLine int // line of 'end' keyword
}

func (s *WhileStmt) Pos() token.Pos { return s.P }
func (*WhileStmt) stmtTag()         {}

func NewWhileStmt(p token.Pos, cond Expr, body *Block, endLine int) *WhileStmt {
	return &WhileStmt{P: p, Cond: cond, Body: body, EndLine: endLine}
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
	P        token.Pos
	EndP     token.Pos // position of the closing 'end' keyword
	Cond     Expr
	ThenLine int // line of 'then' keyword
	Then     *Block
	ElseIfs  []*ElseIf
	Else     *Block // nil if no else
	EndLine  int    // line of 'end' keyword
}

func (s *IfStmt) Pos() token.Pos { return s.P }
func (*IfStmt) stmtTag()         {}

func NewIfStmt(p token.Pos, cond Expr, thenLine int, then *Block, elseifs []*ElseIf, els *Block, endLine int) *IfStmt {
	return &IfStmt{P: p, Cond: cond, ThenLine: thenLine, Then: then, ElseIfs: elseifs, Else: els, EndLine: endLine}
}

// ElseIf is an elseif branch.
type ElseIf struct {
	P        token.Pos
	Cond     Expr
	ThenLine int // line of 'then' keyword
	Then     *Block
}

func (e *ElseIf) Pos() token.Pos { return e.P }

func NewElseIf(p token.Pos, cond Expr, thenLine int, then *Block) *ElseIf {
	return &ElseIf{P: p, Cond: cond, ThenLine: thenLine, Then: then}
}

// ForNumStmt: for name = start, stop [, step] do block end
type ForNumStmt struct {
	P       token.Pos
	EndP    token.Pos // position of the closing 'end' keyword
	Name    *NameExpr
	Start   Expr
	Stop    Expr
	Step    Expr // nil = default 1
	Body    *Block
	EndLine int // line of 'end' keyword
}

func (s *ForNumStmt) Pos() token.Pos { return s.P }
func (*ForNumStmt) stmtTag()         {}

func NewForNumStmt(p token.Pos, name *NameExpr, start, stop, step Expr, body *Block, endLine int) *ForNumStmt {
	return &ForNumStmt{P: p, Name: name, Start: start, Stop: stop, Step: step, Body: body, EndLine: endLine}
}

// ForInStmt: for names in exprs do block end
type ForInStmt struct {
	P       token.Pos
	EndP    token.Pos // position of the closing 'end' keyword
	Names   []*NameExpr
	Iters   []Expr
	Body    *Block
	EndLine int // line of 'end' keyword
}

func (s *ForInStmt) Pos() token.Pos { return s.P }
func (*ForInStmt) stmtTag()         {}

func NewForInStmt(p token.Pos, names []*NameExpr, iters []Expr, body *Block, endLine int) *ForInStmt {
	return &ForInStmt{P: p, Names: names, Iters: iters, Body: body, EndLine: endLine}
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
	P       token.Pos
	Name    string
	EndLine int // line number after closing ::, for duplicate-label error reporting
}

func (s *LabelStmt) Pos() token.Pos { return s.P }
func (*LabelStmt) stmtTag()         {}

func NewLabelStmt(p token.Pos, name string, endLine int) *LabelStmt {
	return &LabelStmt{P: p, Name: name, EndLine: endLine}
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

// ---------------------------------------------------------------------------
// End positions
//
// End returns the position just past a node's last token. Block-closing
// statements (do/while/if/for) carry an explicit EndP set by the parser to the
// 'end' keyword; statements whose tail is an expression derive End from that
// expression. lineFallback yields a line-only position when an explicit EndP
// was not recorded (e.g. nodes built directly in tests).
// ---------------------------------------------------------------------------

func lineFallback(p token.Pos, line int) token.Pos {
	if line == 0 {
		return p
	}
	p.Line = line
	p.Column = 0
	p.Offset = 0
	return p
}

// End of the last expression in a list, or zeroVal if the list is empty.
func lastExprEnd(exprs []Expr) (token.Pos, bool) {
	if n := len(exprs); n > 0 {
		return exprs[n-1].End(), true
	}
	return token.Pos{}, false
}

func (b *Block) End() token.Pos {
	if n := len(b.Stmts); n > 0 {
		return b.Stmts[n-1].End()
	}
	return lineFallback(b.Start, b.EndLine)
}

func (e *ElseIf) End() token.Pos {
	if e.Then != nil {
		return e.Then.End()
	}
	return e.Cond.End()
}

func (s *AssignStmt) End() token.Pos {
	if end, ok := lastExprEnd(s.Values); ok {
		return end
	}
	if end, ok := lastExprEnd(s.Targets); ok {
		return end
	}
	return s.P
}

func (s *LocalStmt) End() token.Pos {
	if end, ok := lastExprEnd(s.Values); ok {
		return end
	}
	if n := len(s.Names); n > 0 {
		return s.Names[n-1].End()
	}
	return s.P
}

func (s *DoStmt) End() token.Pos    { return endOr(s.EndP, lineFallback(s.P, s.EndLine)) }
func (s *WhileStmt) End() token.Pos { return endOr(s.EndP, lineFallback(s.P, s.EndLine)) }
func (s *IfStmt) End() token.Pos    { return endOr(s.EndP, lineFallback(s.P, s.EndLine)) }
func (s *ForNumStmt) End() token.Pos {
	return endOr(s.EndP, lineFallback(s.P, s.EndLine))
}
func (s *ForInStmt) End() token.Pos { return endOr(s.EndP, lineFallback(s.P, s.EndLine)) }

func (s *RepeatStmt) End() token.Pos { return s.Cond.End() }

func (s *ReturnStmt) End() token.Pos {
	if end, ok := lastExprEnd(s.Values); ok {
		return end
	}
	return posAfter(s.P, len("return"))
}

func (s *BreakStmt) End() token.Pos { return posAfter(s.P, len("break")) }

func (s *GotoStmt) End() token.Pos { return posAfter(s.P, len("goto ")+len(s.Label)) }

func (s *LabelStmt) End() token.Pos { return posAfter(s.P, len("::")+len(s.Name)+len("::")) }

func (s *ExprStmt) End() token.Pos { return s.Expr.End() }

func (s *FuncStmt) End() token.Pos {
	if s.Func != nil {
		return s.Func.End()
	}
	return s.Name.End()
}

func (s *LocalFuncStmt) End() token.Pos {
	if s.Func != nil {
		return s.Func.End()
	}
	return s.Name.End()
}

func (s *GlobalStmt) End() token.Pos {
	if end, ok := lastExprEnd(s.Values); ok {
		return end
	}
	if n := len(s.Names); n > 0 {
		return s.Names[n-1].End()
	}
	return s.P
}

func (s *GlobalFuncStmt) End() token.Pos {
	if s.Func != nil {
		return s.Func.End()
	}
	return s.Name.End()
}

func (s *EmptyStmt) End() token.Pos { return posAfter(s.P, 1) }
