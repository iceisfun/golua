package language

import (
	"strings"

	"github.com/iceisfun/golua/ast"
	"github.com/iceisfun/golua/token"
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
		source:    source,
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
	source    string
	lineCount int
	scopes    []*Scope
}

func (a *analyzer) pushScope(parent *Scope, pos token.Pos) *Scope {
	child := &Scope{
		Parent:   parent,
		StartPos: pos,
		EndLine:  parent.EndLine, // will be refined by watermark
	}
	parent.Children = append(parent.Children, child)
	a.scopes = append(a.scopes, child)
	return child
}

// watermark sets the EndLine of scope based on the next statement position.
func (a *analyzer) watermark(scope *Scope, nextPos token.Pos) {
	if nextPos.Line > 0 && nextPos.Line-1 < scope.EndLine {
		scope.EndLine = nextPos.Line - 1
		if scope.EndLine < scope.StartPos.Line {
			scope.EndLine = scope.StartPos.Line
		}
	}
}

func (a *analyzer) walkBlock(block *ast.Block, scope *Scope) {
	if block == nil {
		return
	}
	for i, s := range block.Stmts {
		a.walkStmt(s, scope, block.Stmts, i)
	}
}

func (a *analyzer) walkStmt(s ast.Stmt, scope *Scope, siblings []ast.Stmt, idx int) {
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
		bodyScope := a.pushScope(scope, s.Func.Pos())
		a.addParams(bodyScope, s.Func)
		a.walkBlock(s.Func.Body, bodyScope)
		// Refine end line using next sibling.
		if idx+1 < len(siblings) {
			a.watermark(bodyScope, siblings[idx+1].Pos())
		}

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
		bodyScope := a.pushScope(scope, s.Func.Pos())
		if s.IsMethod {
			bodyScope.Symbols = append(bodyScope.Symbols, &Symbol{
				Name: "self",
				Kind: KindParam,
				Pos:  s.Func.Pos(),
			})
		}
		a.addParams(bodyScope, s.Func)
		a.walkBlock(s.Func.Body, bodyScope)
		if idx+1 < len(siblings) {
			a.watermark(bodyScope, siblings[idx+1].Pos())
		}

	case *ast.GlobalFuncStmt:
		scope.Symbols = append(scope.Symbols, &Symbol{
			Name:     s.Name.Name,
			Kind:     KindFunction,
			Pos:      s.Name.Pos(),
			FuncExpr: s.Func,
			Detail:   funcParams(s.Func),
		})
		bodyScope := a.pushScope(scope, s.Func.Pos())
		a.addParams(bodyScope, s.Func)
		a.walkBlock(s.Func.Body, bodyScope)
		if idx+1 < len(siblings) {
			a.watermark(bodyScope, siblings[idx+1].Pos())
		}

	case *ast.ForNumStmt:
		forScope := a.pushScope(scope, s.Pos())
		forScope.Symbols = append(forScope.Symbols, &Symbol{
			Name: s.Name.Name,
			Kind: KindForVar,
			Pos:  s.Name.Pos(),
		})
		a.walkBlock(s.Body, forScope)
		if idx+1 < len(siblings) {
			a.watermark(forScope, siblings[idx+1].Pos())
		}

	case *ast.ForInStmt:
		forScope := a.pushScope(scope, s.Pos())
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
		if idx+1 < len(siblings) {
			a.watermark(forScope, siblings[idx+1].Pos())
		}

	case *ast.DoStmt:
		doScope := a.pushScope(scope, s.Pos())
		a.walkBlock(s.Body, doScope)
		if idx+1 < len(siblings) {
			a.watermark(doScope, siblings[idx+1].Pos())
		}

	case *ast.WhileStmt:
		a.walkExpr(s.Cond, scope)
		whileScope := a.pushScope(scope, s.Pos())
		a.walkBlock(s.Body, whileScope)
		if idx+1 < len(siblings) {
			a.watermark(whileScope, siblings[idx+1].Pos())
		}

	case *ast.RepeatStmt:
		repeatScope := a.pushScope(scope, s.Pos())
		a.walkBlock(s.Body, repeatScope)
		a.walkExpr(s.Cond, repeatScope) // cond can see body locals
		if idx+1 < len(siblings) {
			a.watermark(repeatScope, siblings[idx+1].Pos())
		}

	case *ast.IfStmt:
		a.walkExpr(s.Cond, scope)
		thenScope := a.pushScope(scope, s.Pos())
		a.walkBlock(s.Then, thenScope)
		if idx+1 < len(siblings) {
			a.watermark(thenScope, siblings[idx+1].Pos())
		}
		for _, elif := range s.ElseIfs {
			a.walkExpr(elif.Cond, scope)
			elifScope := a.pushScope(scope, elif.Pos())
			a.walkBlock(elif.Then, elifScope)
			if idx+1 < len(siblings) {
				a.watermark(elifScope, siblings[idx+1].Pos())
			}
		}
		if s.Else != nil {
			elseScope := a.pushScope(scope, s.Pos())
			a.walkBlock(s.Else, elseScope)
			if idx+1 < len(siblings) {
				a.watermark(elseScope, siblings[idx+1].Pos())
			}
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
		bodyScope := a.pushScope(scope, e.Pos())
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
