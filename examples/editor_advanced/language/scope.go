package language

import (
	"strings"

	"github.com/iceisfun/golua/v2/ast"
	"github.com/iceisfun/golua/v2/token"
)

// SymbolKind describes what kind of symbol this is.
type SymbolKind int

const (
	KindLocal    SymbolKind = iota // local x = ...
	KindParam                      // function parameter
	KindForVar                     // for-loop variable
	KindFunction                   // local function or function statement
	KindGlobal                     // assigned without local
)

func (k SymbolKind) String() string {
	switch k {
	case KindLocal:
		return "local"
	case KindParam:
		return "parameter"
	case KindForVar:
		return "for variable"
	case KindFunction:
		return "function"
	case KindGlobal:
		return "global"
	default:
		return "unknown"
	}
}

// Symbol represents a declared name in a scope.
type Symbol struct {
	Name     string
	Kind     SymbolKind
	Pos      token.Pos
	FuncExpr *ast.FuncExpr // non-nil for functions
	Detail   string        // extra info (e.g. parameter list)
}

// Scope represents a lexical scope in the source.
type Scope struct {
	Parent   *Scope
	Children []*Scope
	Symbols  []*Symbol
	StartPos token.Pos
	EndLine  int // estimated end line via watermark tracking
}

// SymbolTable is the result of scope analysis.
type SymbolTable struct {
	Root   *Scope
	Scopes []*Scope // all scopes in tree order
}

// Analyze walks the AST and builds a symbol table with nested scopes.
func Analyze(block *ast.Block, source string) *SymbolTable {
	a := &analyzer{
		lineCount: strings.Count(source, "\n") + 1,
	}

	root := &Scope{
		StartPos: token.Pos{Line: 1, Column: 1},
		EndLine:  a.lineCount,
	}

	// Pre-populate with stdlib globals.
	for name := range StdlibGlobalNames() {
		root.Symbols = append(root.Symbols, &Symbol{
			Name: name,
			Kind: KindGlobal,
			Pos:  token.Pos{Line: 0, Column: 0},
		})
	}

	a.scopes = append(a.scopes, root)
	a.walkBlock(block, root)

	return &SymbolTable{Root: root, Scopes: a.scopes}
}

// VisibleAt returns all symbols visible at the given 1-based line and column.
func (st *SymbolTable) VisibleAt(line, col int) []*Symbol {
	scope := st.FindScope(line, col)
	if scope == nil {
		scope = st.Root
	}

	seen := make(map[string]bool)
	var result []*Symbol

	for s := scope; s != nil; s = s.Parent {
		for _, sym := range s.Symbols {
			if seen[sym.Name] {
				continue
			}
			// Stdlib symbols (line 0) are always visible.
			// Other symbols must be declared before the cursor position.
			if sym.Pos.Line > 0 && (sym.Pos.Line > line || (sym.Pos.Line == line && sym.Pos.Column > col)) {
				continue
			}
			seen[sym.Name] = true
			result = append(result, sym)
		}
	}
	return result
}

// FindScope returns the innermost scope containing the given position.
func (st *SymbolTable) FindScope(line, col int) *Scope {
	return findScopeIn(st.Root, line, col)
}

func findScopeIn(s *Scope, line, col int) *Scope {
	if !scopeContains(s, line) {
		return nil
	}
	// Try children (innermost first).
	for _, child := range s.Children {
		if found := findScopeIn(child, line, col); found != nil {
			return found
		}
	}
	return s
}

func scopeContains(s *Scope, line int) bool {
	start := s.StartPos.Line
	if start == 0 {
		start = 1
	}
	return line >= start && line <= s.EndLine
}

type analyzer struct {
	lineCount int
	scopes    []*Scope
}

// pushScope creates a child scope spanning [pos.Line, endLine]. The end line
// comes from the AST node's End() span (the closing keyword / function body),
// which the parser now tracks accurately — far more reliable than the old
// "next sibling minus one" watermark heuristic, which mis-bounded the last
// statement in a block and every if/elseif/else branch.
func (a *analyzer) pushScope(parent *Scope, pos token.Pos, endLine int) *Scope {
	if endLine <= 0 || endLine > parent.EndLine {
		endLine = parent.EndLine // fall back to the enclosing scope's bound
	}
	if endLine < pos.Line {
		endLine = pos.Line
	}
	child := &Scope{
		Parent:   parent,
		StartPos: pos,
		EndLine:  endLine,
	}
	parent.Children = append(parent.Children, child)
	a.scopes = append(a.scopes, child)
	return child
}

func (a *analyzer) walkBlock(block *ast.Block, scope *Scope) {
	if block == nil {
		return
	}
	for _, s := range block.Stmts {
		a.walkStmt(s, scope)
	}
}

// endLineOf returns the 1-based line of a node's closing span, or 0 if unknown.
func endLineOf(n ast.Node) int {
	if n == nil {
		return 0
	}
	return n.End().Line
}

func (a *analyzer) walkStmt(s ast.Stmt, scope *Scope) {
	switch s := s.(type) {
	case *ast.LocalStmt:
		for i, name := range s.Names {
			sym := &Symbol{
				Name: name.Name,
				Kind: KindLocal,
				Pos:  name.Pos(),
			}
			// Check if value is a FuncExpr.
			if i < len(s.Values) {
				if fe, ok := s.Values[i].(*ast.FuncExpr); ok {
					sym.Kind = KindFunction
					sym.FuncExpr = fe
					sym.Detail = funcParams(fe)
				}
			}
			scope.Symbols = append(scope.Symbols, sym)
		}
		// Walk values for nested expressions.
		for _, v := range s.Values {
			a.walkExpr(v, scope)
		}

	case *ast.LocalFuncStmt:
		scope.Symbols = append(scope.Symbols, &Symbol{
			Name:     s.Name.Name,
			Kind:     KindFunction,
			Pos:      s.Name.Pos(),
			FuncExpr: s.Func,
			Detail:   funcParams(s.Func),
		})
		// Function body gets its own scope with params.
		bodyScope := a.pushScope(scope, s.Func.Pos(), endLineOf(s.Func))
		a.addParams(bodyScope, s.Func)
		a.walkBlock(s.Func.Body, bodyScope)

	case *ast.FuncStmt:
		// Record as global function.
		name := funcStmtName(s)
		if name != "" {
			scope.Symbols = append(scope.Symbols, &Symbol{
				Name:     name,
				Kind:     KindFunction,
				Pos:      s.Pos(),
				FuncExpr: s.Func,
				Detail:   funcParams(s.Func),
			})
		}
		// Function body scope.
		bodyScope := a.pushScope(scope, s.Func.Pos(), endLineOf(s.Func))
		if s.IsMethod {
			bodyScope.Symbols = append(bodyScope.Symbols, &Symbol{
				Name: "self",
				Kind: KindParam,
				Pos:  s.Func.Pos(),
			})
		}
		a.addParams(bodyScope, s.Func)
		a.walkBlock(s.Func.Body, bodyScope)

	case *ast.GlobalFuncStmt:
		scope.Symbols = append(scope.Symbols, &Symbol{
			Name:     s.Name.Name,
			Kind:     KindFunction,
			Pos:      s.Name.Pos(),
			FuncExpr: s.Func,
			Detail:   funcParams(s.Func),
		})
		bodyScope := a.pushScope(scope, s.Func.Pos(), endLineOf(s.Func))
		a.addParams(bodyScope, s.Func)
		a.walkBlock(s.Func.Body, bodyScope)

	case *ast.ForNumStmt:
		forScope := a.pushScope(scope, s.Pos(), endLineOf(s))
		forScope.Symbols = append(forScope.Symbols, &Symbol{
			Name: s.Name.Name,
			Kind: KindForVar,
			Pos:  s.Name.Pos(),
		})
		a.walkBlock(s.Body, forScope)

	case *ast.ForInStmt:
		forScope := a.pushScope(scope, s.Pos(), endLineOf(s))
		for _, name := range s.Names {
			forScope.Symbols = append(forScope.Symbols, &Symbol{
				Name: name.Name,
				Kind: KindForVar,
				Pos:  name.Pos(),
			})
		}
		// Walk iterators in parent scope.
		for _, iter := range s.Iters {
			a.walkExpr(iter, scope)
		}
		a.walkBlock(s.Body, forScope)

	case *ast.DoStmt:
		doScope := a.pushScope(scope, s.Pos(), endLineOf(s))
		a.walkBlock(s.Body, doScope)

	case *ast.WhileStmt:
		a.walkExpr(s.Cond, scope)
		whileScope := a.pushScope(scope, s.Pos(), endLineOf(s))
		a.walkBlock(s.Body, whileScope)

	case *ast.RepeatStmt:
		repeatScope := a.pushScope(scope, s.Pos(), endLineOf(s))
		a.walkBlock(s.Body, repeatScope)
		a.walkExpr(s.Cond, repeatScope) // cond can see body locals

	case *ast.IfStmt:
		a.walkExpr(s.Cond, scope)
		// Each branch is its own scope, bounded by that branch's block span.
		thenScope := a.pushScope(scope, s.Pos(), endLineOf(s.Then))
		a.walkBlock(s.Then, thenScope)
		for _, elif := range s.ElseIfs {
			a.walkExpr(elif.Cond, scope)
			elifScope := a.pushScope(scope, elif.Pos(), endLineOf(elif.Then))
			a.walkBlock(elif.Then, elifScope)
		}
		if s.Else != nil {
			elseScope := a.pushScope(scope, s.Else.Pos(), endLineOf(s.Else))
			a.walkBlock(s.Else, elseScope)
		}

	case *ast.AssignStmt:
		// Detect global assignments.
		for _, t := range s.Targets {
			if ne, ok := t.(*ast.NameExpr); ok {
				if !a.isLocal(ne.Name, scope) {
					scope.Symbols = append(scope.Symbols, &Symbol{
						Name: ne.Name,
						Kind: KindGlobal,
						Pos:  ne.Pos(),
					})
				}
			}
		}
		for _, v := range s.Values {
			a.walkExpr(v, scope)
		}

	case *ast.GlobalStmt:
		for _, name := range s.Names {
			scope.Symbols = append(scope.Symbols, &Symbol{
				Name: name.Name,
				Kind: KindGlobal,
				Pos:  name.Pos(),
			})
		}
		for _, v := range s.Values {
			a.walkExpr(v, scope)
		}

	case *ast.ReturnStmt:
		for _, v := range s.Values {
			a.walkExpr(v, scope)
		}

	case *ast.ExprStmt:
		a.walkExpr(s.Expr, scope)

		// BreakStmt, GotoStmt, LabelStmt, EmptyStmt — no scope effects.
	}
}

func (a *analyzer) walkExpr(e ast.Expr, scope *Scope) {
	if e == nil {
		return
	}
	switch e := e.(type) {
	case *ast.FuncExpr:
		bodyScope := a.pushScope(scope, e.Pos(), endLineOf(e))
		a.addParams(bodyScope, e)
		a.walkBlock(e.Body, bodyScope)

	case *ast.BinopExpr:
		a.walkExpr(e.Left, scope)
		a.walkExpr(e.Right, scope)

	case *ast.UnopExpr:
		a.walkExpr(e.Operand, scope)

	case *ast.FuncCallExpr:
		a.walkExpr(e.Func, scope)
		for _, arg := range e.Args {
			a.walkExpr(arg, scope)
		}

	case *ast.MethodCallExpr:
		a.walkExpr(e.Object, scope)
		for _, arg := range e.Args {
			a.walkExpr(arg, scope)
		}

	case *ast.IndexExpr:
		a.walkExpr(e.Table, scope)
		a.walkExpr(e.Key, scope)

	case *ast.FieldExpr:
		a.walkExpr(e.Table, scope)

	case *ast.TableConstructor:
		for _, f := range e.Fields {
			if f.Key != nil {
				a.walkExpr(f.Key, scope)
			}
			a.walkExpr(f.Value, scope)
		}

	case *ast.ParenExpr:
		a.walkExpr(e.Inner, scope)
	}
}

func (a *analyzer) addParams(scope *Scope, fe *ast.FuncExpr) {
	for _, p := range fe.Params {
		scope.Symbols = append(scope.Symbols, &Symbol{
			Name: p.Name,
			Kind: KindParam,
			Pos:  p.Pos(),
		})
	}
}

func (a *analyzer) isLocal(name string, scope *Scope) bool {
	for s := scope; s != nil; s = s.Parent {
		for _, sym := range s.Symbols {
			if sym.Name == name && sym.Kind != KindGlobal {
				return true
			}
		}
	}
	return false
}

func funcParams(fe *ast.FuncExpr) string {
	if fe == nil {
		return ""
	}
	var parts []string
	for _, p := range fe.Params {
		parts = append(parts, p.Name)
	}
	if fe.VarArg {
		parts = append(parts, "...")
	}
	return strings.Join(parts, ", ")
}

func funcStmtName(s *ast.FuncStmt) string {
	switch n := s.Name.(type) {
	case *ast.NameExpr:
		return n.Name
	case *ast.FieldExpr:
		// e.g. table.method — return "table.method"
		if ne, ok := n.Table.(*ast.NameExpr); ok {
			return ne.Name + "." + n.Field
		}
	}
	return ""
}
