package ast_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/iceisfun/golua/v2/ast"
	"github.com/iceisfun/golua/v2/parser"
	"github.com/iceisfun/golua/v2/token"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func pos(line, col int) token.Pos {
	return token.Pos{Source: "test", Line: line, Column: col}
}

func p() token.Pos { return pos(1, 1) }

func emptyBlock() *ast.Block {
	return &ast.Block{Start: p()}
}

func blockWith(stmts ...ast.Stmt) *ast.Block {
	return &ast.Block{Start: p(), Stmts: stmts}
}

func parseDump(t *testing.T, src string) string {
	t.Helper()
	block, err := parser.Parse("test", src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	return ast.DumpString(block)
}

func mustContain(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Errorf("output does not contain %q\ngot:\n%s", want, got)
	}
}

func mustNotContain(t *testing.T, got, unwanted string) {
	t.Helper()
	if strings.Contains(got, unwanted) {
		t.Errorf("output should not contain %q\ngot:\n%s", unwanted, got)
	}
}

// ---------------------------------------------------------------------------
// Constructor + Pos() tests for every node type
// ---------------------------------------------------------------------------

func TestBlockPos(t *testing.T) {
	b := &ast.Block{Start: pos(5, 3)}
	if b.Pos().Line != 5 || b.Pos().Column != 3 {
		t.Errorf("Block.Pos() = %v, want line=5 col=3", b.Pos())
	}
}

func TestExpressionConstructors(t *testing.T) {
	tests := []struct {
		name string
		node ast.Expr
		line int
		col  int
	}{
		{"NilExpr", ast.NewNilExpr(pos(1, 1)), 1, 1},
		{"TrueExpr", ast.NewTrueExpr(pos(2, 3)), 2, 3},
		{"FalseExpr", ast.NewFalseExpr(pos(3, 5)), 3, 5},
		{"NumberExpr", ast.NewNumberExpr(pos(4, 1), 42, "42"), 4, 1},
		{"FloatExpr", ast.NewFloatExpr(pos(5, 1), 3.14, "3.14"), 5, 1},
		{"StringExpr", ast.NewStringExpr(pos(6, 1), "hello"), 6, 1},
		{"VarArgExpr", ast.NewVarArgExpr(pos(7, 1)), 7, 1},
		{"NameExpr", ast.NewNameExpr(pos(8, 1), "x"), 8, 1},
		{"BinopExpr", ast.NewBinopExpr(pos(9, 1), "+",
			ast.NewNumberExpr(p(), 1, "1"),
			ast.NewNumberExpr(p(), 2, "2")), 9, 1},
		{"UnopExpr", ast.NewUnopExpr(pos(10, 1), "-",
			ast.NewNumberExpr(p(), 1, "1")), 10, 1},
		{"IndexExpr", ast.NewIndexExpr(pos(11, 1),
			ast.NewNameExpr(p(), "t"),
			ast.NewNumberExpr(p(), 1, "1")), 11, 1},
		{"FieldExpr", ast.NewFieldExpr(pos(12, 1),
			ast.NewNameExpr(p(), "t"), "x"), 12, 1},
		{"MethodCallExpr", ast.NewMethodCallExpr(pos(13, 1),
			ast.NewNameExpr(p(), "obj"), "foo",
			[]ast.Expr{ast.NewNumberExpr(p(), 1, "1")}), 13, 1},
		{"FuncCallExpr", ast.NewFuncCallExpr(pos(14, 1),
			ast.NewNameExpr(p(), "f"),
			[]ast.Expr{ast.NewNumberExpr(p(), 1, "1")}), 14, 1},
		{"FuncExpr", ast.NewFuncExpr(pos(15, 1),
			[]*ast.NameExpr{ast.NewNameExpr(p(), "a")},
			false, "", emptyBlock()), 15, 1},
		{"TableConstructor", ast.NewTableConstructor(pos(16, 1), nil), 16, 1},
		{"ParenExpr", ast.NewParenExpr(pos(17, 1),
			ast.NewNumberExpr(p(), 1, "1")), 17, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.node.Pos()
			if got.Line != tt.line || got.Column != tt.col {
				t.Errorf("%s.Pos() = %v, want line=%d col=%d", tt.name, got, tt.line, tt.col)
			}
		})
	}
}

func TestTableFieldPos(t *testing.T) {
	tf := ast.NewTableField(pos(3, 7),
		ast.NewStringExpr(p(), "k"),
		ast.NewNumberExpr(p(), 1, "1"))
	if tf.Pos().Line != 3 || tf.Pos().Column != 7 {
		t.Errorf("TableField.Pos() = %v, want line=3 col=7", tf.Pos())
	}
}

func TestElseIfPos(t *testing.T) {
	ei := ast.NewElseIf(pos(10, 2), ast.NewTrueExpr(p()), 0, emptyBlock())
	if ei.Pos().Line != 10 || ei.Pos().Column != 2 {
		t.Errorf("ElseIf.Pos() = %v, want line=10 col=2", ei.Pos())
	}
}

func TestStatementConstructors(t *testing.T) {
	tests := []struct {
		name string
		node ast.Stmt
		line int
		col  int
	}{
		{"AssignStmt", ast.NewAssignStmt(pos(1, 1),
			[]ast.Expr{ast.NewNameExpr(p(), "x")},
			[]ast.Expr{ast.NewNumberExpr(p(), 1, "1")}), 1, 1},
		{"LocalStmt", ast.NewLocalStmt(pos(2, 1),
			[]*ast.NameExpr{ast.NewNameExpr(p(), "x")},
			[]string{""}, []ast.Expr{ast.NewNumberExpr(p(), 1, "1")}), 2, 1},
		{"DoStmt", ast.NewDoStmt(pos(3, 1), emptyBlock(), 3), 3, 1},
		{"WhileStmt", ast.NewWhileStmt(pos(4, 1),
			ast.NewTrueExpr(p()), emptyBlock(), 0), 4, 1},
		{"RepeatStmt", ast.NewRepeatStmt(pos(5, 1),
			emptyBlock(), ast.NewTrueExpr(p())), 5, 1},
		{"IfStmt", ast.NewIfStmt(pos(6, 1),
			ast.NewTrueExpr(p()), 0, emptyBlock(), nil, nil, 0), 6, 1},
		{"ForNumStmt", ast.NewForNumStmt(pos(7, 1),
			ast.NewNameExpr(p(), "i"),
			ast.NewNumberExpr(p(), 1, "1"),
			ast.NewNumberExpr(p(), 10, "10"),
			nil, emptyBlock(), 0), 7, 1},
		{"ForInStmt", ast.NewForInStmt(pos(8, 1),
			[]*ast.NameExpr{ast.NewNameExpr(p(), "k")},
			[]ast.Expr{ast.NewNameExpr(p(), "pairs")},
			emptyBlock(), 0), 8, 1},
		{"ReturnStmt", ast.NewReturnStmt(pos(9, 1), nil), 9, 1},
		{"BreakStmt", ast.NewBreakStmt(pos(10, 1)), 10, 1},
		{"GotoStmt", ast.NewGotoStmt(pos(11, 1), "skip"), 11, 1},
		{"LabelStmt", ast.NewLabelStmt(pos(12, 1), "skip", 12), 12, 1},
		{"ExprStmt", ast.NewExprStmt(pos(13, 1),
			ast.NewFuncCallExpr(p(), ast.NewNameExpr(p(), "f"), nil)), 13, 1},
		{"FuncStmt", ast.NewFuncStmt(pos(14, 1),
			ast.NewNameExpr(p(), "f"), false,
			ast.NewFuncExpr(p(), nil, false, "", emptyBlock())), 14, 1},
		{"LocalFuncStmt", ast.NewLocalFuncStmt(pos(15, 1),
			ast.NewNameExpr(p(), "f"),
			ast.NewFuncExpr(p(), nil, false, "", emptyBlock())), 15, 1},
		{"GlobalStmt", ast.NewGlobalStmt(pos(16, 1),
			[]*ast.NameExpr{ast.NewNameExpr(p(), "x")},
			[]string{""}, nil), 16, 1},
		{"GlobalFuncStmt", ast.NewGlobalFuncStmt(pos(17, 1),
			ast.NewNameExpr(p(), "f"),
			ast.NewFuncExpr(p(), nil, false, "", emptyBlock())), 17, 1},
		{"EmptyStmt", ast.NewEmptyStmt(pos(18, 1)), 18, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.node.Pos()
			if got.Line != tt.line || got.Column != tt.col {
				t.Errorf("%s.Pos() = %v, want line=%d col=%d", tt.name, got, tt.line, tt.col)
			}
		})
	}
}

func TestGlobalStarStmt(t *testing.T) {
	g := ast.NewGlobalStarStmt(pos(1, 1), "const")
	if !g.Star {
		t.Error("expected Star=true")
	}
	if len(g.Attribs) != 1 || g.Attribs[0] != "const" {
		t.Errorf("Attribs = %v, want [const]", g.Attribs)
	}
}

// ---------------------------------------------------------------------------
// DumpString tests — manually constructed ASTs
// ---------------------------------------------------------------------------

func TestDumpEmptyBlock(t *testing.T) {
	d := ast.DumpString(emptyBlock())
	mustContain(t, d, "(block)")
}

func TestDumpNonEmptyBlock(t *testing.T) {
	b := blockWith(ast.NewReturnStmt(p(), nil))
	d := ast.DumpString(b)
	mustContain(t, d, "(block")
	mustContain(t, d, "(return)")
}

func TestDumpNilBlock(t *testing.T) {
	d := ast.DumpString(nil)
	mustContain(t, d, "(block)")
}

func TestDumpExprNil(t *testing.T) {
	b := blockWith(ast.NewReturnStmt(p(), []ast.Expr{ast.NewNilExpr(p())}))
	d := ast.DumpString(b)
	mustContain(t, d, "nil")
}

func TestDumpExprTrue(t *testing.T) {
	b := blockWith(ast.NewReturnStmt(p(), []ast.Expr{ast.NewTrueExpr(p())}))
	d := ast.DumpString(b)
	mustContain(t, d, "true")
}

func TestDumpExprFalse(t *testing.T) {
	b := blockWith(ast.NewReturnStmt(p(), []ast.Expr{ast.NewFalseExpr(p())}))
	d := ast.DumpString(b)
	mustContain(t, d, "false")
}

func TestDumpExprNumber(t *testing.T) {
	b := blockWith(ast.NewReturnStmt(p(), []ast.Expr{
		ast.NewNumberExpr(p(), 42, "42"),
	}))
	d := ast.DumpString(b)
	mustContain(t, d, "(int 42)")
}

func TestDumpExprFloat(t *testing.T) {
	b := blockWith(ast.NewReturnStmt(p(), []ast.Expr{
		ast.NewFloatExpr(p(), 3.14, "3.14"),
	}))
	d := ast.DumpString(b)
	mustContain(t, d, "(float 3.14)")
}

func TestDumpExprString(t *testing.T) {
	b := blockWith(ast.NewReturnStmt(p(), []ast.Expr{
		ast.NewStringExpr(p(), "hello world"),
	}))
	d := ast.DumpString(b)
	mustContain(t, d, `(string "hello world")`)
}

func TestDumpExprVarArg(t *testing.T) {
	b := blockWith(ast.NewReturnStmt(p(), []ast.Expr{
		ast.NewVarArgExpr(p()),
	}))
	d := ast.DumpString(b)
	mustContain(t, d, "...")
}

func TestDumpExprName(t *testing.T) {
	b := blockWith(ast.NewReturnStmt(p(), []ast.Expr{
		ast.NewNameExpr(p(), "myvar"),
	}))
	d := ast.DumpString(b)
	mustContain(t, d, "(name myvar)")
}

func TestDumpExprBinop(t *testing.T) {
	b := blockWith(ast.NewReturnStmt(p(), []ast.Expr{
		ast.NewBinopExpr(p(), "+",
			ast.NewNumberExpr(p(), 1, "1"),
			ast.NewNumberExpr(p(), 2, "2")),
	}))
	d := ast.DumpString(b)
	mustContain(t, d, "(binop +")
	mustContain(t, d, "(int 1)")
	mustContain(t, d, "(int 2)")
}

func TestDumpExprUnop(t *testing.T) {
	b := blockWith(ast.NewReturnStmt(p(), []ast.Expr{
		ast.NewUnopExpr(p(), "-", ast.NewNumberExpr(p(), 5, "5")),
	}))
	d := ast.DumpString(b)
	mustContain(t, d, "(unop -")
	mustContain(t, d, "(int 5)")
}

func TestDumpExprIndex(t *testing.T) {
	b := blockWith(ast.NewReturnStmt(p(), []ast.Expr{
		ast.NewIndexExpr(p(),
			ast.NewNameExpr(p(), "t"),
			ast.NewNumberExpr(p(), 1, "1")),
	}))
	d := ast.DumpString(b)
	mustContain(t, d, "(index")
	mustContain(t, d, "(name t)")
	mustContain(t, d, "(int 1)")
}

func TestDumpExprField(t *testing.T) {
	b := blockWith(ast.NewReturnStmt(p(), []ast.Expr{
		ast.NewFieldExpr(p(), ast.NewNameExpr(p(), "obj"), "name"),
	}))
	d := ast.DumpString(b)
	mustContain(t, d, "(field .name")
	mustContain(t, d, "(name obj)")
}

func TestDumpExprMethodCall(t *testing.T) {
	b := blockWith(ast.NewExprStmt(p(),
		ast.NewMethodCallExpr(p(),
			ast.NewNameExpr(p(), "obj"), "method",
			[]ast.Expr{ast.NewNumberExpr(p(), 1, "1")}),
	))
	d := ast.DumpString(b)
	mustContain(t, d, "(method-call :method")
	mustContain(t, d, "(name obj)")
	mustContain(t, d, "(args")
	mustContain(t, d, "(int 1)")
}

func TestDumpExprMethodCallNoArgs(t *testing.T) {
	b := blockWith(ast.NewExprStmt(p(),
		ast.NewMethodCallExpr(p(),
			ast.NewNameExpr(p(), "obj"), "done", nil),
	))
	d := ast.DumpString(b)
	mustContain(t, d, "(method-call :done")
	mustNotContain(t, d, "(args")
}

func TestDumpExprFuncCall(t *testing.T) {
	b := blockWith(ast.NewExprStmt(p(),
		ast.NewFuncCallExpr(p(),
			ast.NewNameExpr(p(), "print"),
			[]ast.Expr{ast.NewStringExpr(p(), "hi")}),
	))
	d := ast.DumpString(b)
	mustContain(t, d, "(call")
	mustContain(t, d, "(name print)")
	mustContain(t, d, "(args")
	mustContain(t, d, `(string "hi")`)
}

func TestDumpExprFuncCallNoArgs(t *testing.T) {
	b := blockWith(ast.NewExprStmt(p(),
		ast.NewFuncCallExpr(p(),
			ast.NewNameExpr(p(), "noop"), nil),
	))
	d := ast.DumpString(b)
	mustContain(t, d, "(call")
	mustNotContain(t, d, "(args")
}

func TestDumpExprFuncExpr(t *testing.T) {
	fe := ast.NewFuncExpr(p(),
		[]*ast.NameExpr{
			ast.NewNameExpr(p(), "a"),
			ast.NewNameExpr(p(), "b"),
		},
		false, "",
		blockWith(ast.NewReturnStmt(p(), []ast.Expr{
			ast.NewBinopExpr(p(), "+",
				ast.NewNameExpr(p(), "a"),
				ast.NewNameExpr(p(), "b")),
		})))
	b := blockWith(ast.NewReturnStmt(p(), []ast.Expr{fe}))
	d := ast.DumpString(b)
	mustContain(t, d, "(function (a, b)")
	mustContain(t, d, "(binop +")
}

func TestDumpExprFuncExprVararg(t *testing.T) {
	fe := ast.NewFuncExpr(p(),
		[]*ast.NameExpr{ast.NewNameExpr(p(), "x")},
		true, "", emptyBlock())
	b := blockWith(ast.NewReturnStmt(p(), []ast.Expr{fe}))
	d := ast.DumpString(b)
	mustContain(t, d, "(function (x, ...)")
}

func TestDumpExprFuncExprVarargOnly(t *testing.T) {
	fe := ast.NewFuncExpr(p(), nil, true, "", emptyBlock())
	b := blockWith(ast.NewReturnStmt(p(), []ast.Expr{fe}))
	d := ast.DumpString(b)
	mustContain(t, d, "(function (...)")
}

func TestDumpExprFuncExprVarargName(t *testing.T) {
	fe := ast.NewFuncExpr(p(), nil, true, "args", emptyBlock())
	b := blockWith(ast.NewReturnStmt(p(), []ast.Expr{fe}))
	d := ast.DumpString(b)
	mustContain(t, d, "(function (...args)")
}

func TestDumpExprFuncExprVarargNameWithParams(t *testing.T) {
	fe := ast.NewFuncExpr(p(),
		[]*ast.NameExpr{ast.NewNameExpr(p(), "a")},
		true, "rest", emptyBlock())
	b := blockWith(ast.NewReturnStmt(p(), []ast.Expr{fe}))
	d := ast.DumpString(b)
	mustContain(t, d, "(function (a, ...rest)")
}

func TestDumpExprTableEmpty(t *testing.T) {
	tc := ast.NewTableConstructor(p(), nil)
	b := blockWith(ast.NewReturnStmt(p(), []ast.Expr{tc}))
	d := ast.DumpString(b)
	mustContain(t, d, "(table {})")
}

func TestDumpExprTablePositionalItems(t *testing.T) {
	tc := ast.NewTableConstructor(p(), []*ast.TableField{
		ast.NewTableField(p(), nil, ast.NewNumberExpr(p(), 1, "1")),
		ast.NewTableField(p(), nil, ast.NewNumberExpr(p(), 2, "2")),
	})
	b := blockWith(ast.NewReturnStmt(p(), []ast.Expr{tc}))
	d := ast.DumpString(b)
	mustContain(t, d, "(table {")
	mustContain(t, d, "(item")
	mustContain(t, d, "(int 1)")
	mustContain(t, d, "(int 2)")
	mustContain(t, d, "})")
}

func TestDumpExprTableKeyedFields(t *testing.T) {
	tc := ast.NewTableConstructor(p(), []*ast.TableField{
		ast.NewTableField(p(),
			ast.NewStringExpr(p(), "x"),
			ast.NewNumberExpr(p(), 10, "10")),
	})
	b := blockWith(ast.NewReturnStmt(p(), []ast.Expr{tc}))
	d := ast.DumpString(b)
	mustContain(t, d, "(field")
	mustContain(t, d, "(key")
	mustContain(t, d, `(string "x")`)
	mustContain(t, d, "(val")
	mustContain(t, d, "(int 10)")
}

func TestDumpExprTableMixed(t *testing.T) {
	tc := ast.NewTableConstructor(p(), []*ast.TableField{
		ast.NewTableField(p(), nil, ast.NewNumberExpr(p(), 1, "1")),
		ast.NewTableField(p(),
			ast.NewStringExpr(p(), "k"),
			ast.NewNumberExpr(p(), 2, "2")),
	})
	b := blockWith(ast.NewReturnStmt(p(), []ast.Expr{tc}))
	d := ast.DumpString(b)
	mustContain(t, d, "(item")
	mustContain(t, d, "(key")
}

func TestDumpExprParen(t *testing.T) {
	b := blockWith(ast.NewReturnStmt(p(), []ast.Expr{
		ast.NewParenExpr(p(), ast.NewNumberExpr(p(), 42, "42")),
	}))
	d := ast.DumpString(b)
	mustContain(t, d, "(paren")
	mustContain(t, d, "(int 42)")
}

// ---------------------------------------------------------------------------
// DumpString tests — statements (manually constructed)
// ---------------------------------------------------------------------------

func TestDumpStmtAssign(t *testing.T) {
	s := ast.NewAssignStmt(p(),
		[]ast.Expr{ast.NewNameExpr(p(), "x")},
		[]ast.Expr{ast.NewNumberExpr(p(), 1, "1")})
	d := ast.DumpString(blockWith(s))
	mustContain(t, d, "(assign")
	mustContain(t, d, "(targets")
	mustContain(t, d, "(name x)")
	mustContain(t, d, "(values")
	mustContain(t, d, "(int 1)")
}

func TestDumpStmtLocal(t *testing.T) {
	s := ast.NewLocalStmt(p(),
		[]*ast.NameExpr{ast.NewNameExpr(p(), "a"), ast.NewNameExpr(p(), "b")},
		[]string{"", ""},
		[]ast.Expr{ast.NewNumberExpr(p(), 1, "1"), ast.NewNumberExpr(p(), 2, "2")})
	d := ast.DumpString(blockWith(s))
	mustContain(t, d, "(local")
	mustContain(t, d, "(names")
	mustContain(t, d, "a")
	mustContain(t, d, "b")
	mustContain(t, d, "(values")
}

func TestDumpStmtLocalNoValues(t *testing.T) {
	s := ast.NewLocalStmt(p(),
		[]*ast.NameExpr{ast.NewNameExpr(p(), "x")},
		[]string{""}, nil)
	d := ast.DumpString(blockWith(s))
	mustContain(t, d, "(local")
	mustNotContain(t, d, "(values")
}

func TestDumpStmtLocalConst(t *testing.T) {
	s := ast.NewLocalStmt(p(),
		[]*ast.NameExpr{ast.NewNameExpr(p(), "pi")},
		[]string{"const"},
		[]ast.Expr{ast.NewFloatExpr(p(), 3.14, "3.14")})
	d := ast.DumpString(blockWith(s))
	mustContain(t, d, "pi <const>")
}

func TestDumpStmtLocalClose(t *testing.T) {
	s := ast.NewLocalStmt(p(),
		[]*ast.NameExpr{ast.NewNameExpr(p(), "f")},
		[]string{"close"},
		[]ast.Expr{ast.NewNameExpr(p(), "handle")})
	d := ast.DumpString(blockWith(s))
	mustContain(t, d, "f <close>")
}

func TestDumpStmtDo(t *testing.T) {
	inner := blockWith(ast.NewReturnStmt(p(), nil))
	s := ast.NewDoStmt(p(), inner, 0)
	d := ast.DumpString(blockWith(s))
	mustContain(t, d, "(do")
	mustContain(t, d, "(return)")
}

func TestDumpStmtWhile(t *testing.T) {
	s := ast.NewWhileStmt(p(), ast.NewTrueExpr(p()),
		blockWith(ast.NewBreakStmt(p())), 0)
	d := ast.DumpString(blockWith(s))
	mustContain(t, d, "(while")
	mustContain(t, d, "true")
	mustContain(t, d, "(break)")
}

func TestDumpStmtRepeat(t *testing.T) {
	s := ast.NewRepeatStmt(p(),
		blockWith(ast.NewBreakStmt(p())),
		ast.NewTrueExpr(p()))
	d := ast.DumpString(blockWith(s))
	mustContain(t, d, "(repeat")
	mustContain(t, d, "(until")
	mustContain(t, d, "true")
}

func TestDumpStmtIf(t *testing.T) {
	s := ast.NewIfStmt(p(),
		ast.NewNameExpr(p(), "cond"), 0,
		blockWith(ast.NewReturnStmt(p(), []ast.Expr{ast.NewNumberExpr(p(), 1, "1")})),
		nil, nil, 0)
	d := ast.DumpString(blockWith(s))
	mustContain(t, d, "(if")
	mustContain(t, d, "(name cond)")
	mustContain(t, d, "(then")
}

func TestDumpStmtIfElseIf(t *testing.T) {
	s := ast.NewIfStmt(p(),
		ast.NewNameExpr(p(), "a"), 0,
		emptyBlock(),
		[]*ast.ElseIf{
			ast.NewElseIf(p(), ast.NewNameExpr(p(), "b"), 0, emptyBlock()),
		},
		nil, 0)
	d := ast.DumpString(blockWith(s))
	mustContain(t, d, "(if")
	mustContain(t, d, "(elseif")
	mustContain(t, d, "(name b)")
}

func TestDumpStmtIfElse(t *testing.T) {
	s := ast.NewIfStmt(p(),
		ast.NewTrueExpr(p()), 0,
		emptyBlock(), nil,
		blockWith(ast.NewReturnStmt(p(), nil)), 0)
	d := ast.DumpString(blockWith(s))
	mustContain(t, d, "(if")
	mustContain(t, d, "(else")
}

func TestDumpStmtIfElseIfElse(t *testing.T) {
	s := ast.NewIfStmt(p(),
		ast.NewNameExpr(p(), "a"), 0,
		emptyBlock(),
		[]*ast.ElseIf{
			ast.NewElseIf(p(), ast.NewNameExpr(p(), "b"), 0, emptyBlock()),
			ast.NewElseIf(p(), ast.NewNameExpr(p(), "c"), 0, emptyBlock()),
		},
		blockWith(ast.NewReturnStmt(p(), nil)), 0)
	d := ast.DumpString(blockWith(s))
	mustContain(t, d, "(if")
	mustContain(t, d, "(elseif")
	mustContain(t, d, "(name b)")
	mustContain(t, d, "(name c)")
	mustContain(t, d, "(else")
}

func TestDumpStmtForNum(t *testing.T) {
	s := ast.NewForNumStmt(p(),
		ast.NewNameExpr(p(), "i"),
		ast.NewNumberExpr(p(), 1, "1"),
		ast.NewNumberExpr(p(), 10, "10"),
		nil, emptyBlock(), 0)
	d := ast.DumpString(blockWith(s))
	mustContain(t, d, "(for-num i")
	mustContain(t, d, "(start")
	mustContain(t, d, "(stop")
	mustNotContain(t, d, "(step")
}

func TestDumpStmtForNumWithStep(t *testing.T) {
	s := ast.NewForNumStmt(p(),
		ast.NewNameExpr(p(), "i"),
		ast.NewNumberExpr(p(), 1, "1"),
		ast.NewNumberExpr(p(), 100, "100"),
		ast.NewNumberExpr(p(), 2, "2"),
		emptyBlock(), 0)
	d := ast.DumpString(blockWith(s))
	mustContain(t, d, "(for-num i")
	mustContain(t, d, "(step")
	mustContain(t, d, "(int 2)")
}

func TestDumpStmtForIn(t *testing.T) {
	s := ast.NewForInStmt(p(),
		[]*ast.NameExpr{
			ast.NewNameExpr(p(), "k"),
			ast.NewNameExpr(p(), "v"),
		},
		[]ast.Expr{
			ast.NewFuncCallExpr(p(),
				ast.NewNameExpr(p(), "pairs"),
				[]ast.Expr{ast.NewNameExpr(p(), "t")}),
		},
		emptyBlock(), 0)
	d := ast.DumpString(blockWith(s))
	mustContain(t, d, "(for-in [k, v]")
	mustContain(t, d, "(iters")
}

func TestDumpStmtReturnEmpty(t *testing.T) {
	d := ast.DumpString(blockWith(ast.NewReturnStmt(p(), nil)))
	mustContain(t, d, "(return)")
}

func TestDumpStmtReturnValues(t *testing.T) {
	d := ast.DumpString(blockWith(ast.NewReturnStmt(p(), []ast.Expr{
		ast.NewNumberExpr(p(), 1, "1"),
		ast.NewNumberExpr(p(), 2, "2"),
	})))
	mustContain(t, d, "(return")
	mustContain(t, d, "(int 1)")
	mustContain(t, d, "(int 2)")
}

func TestDumpStmtBreak(t *testing.T) {
	d := ast.DumpString(blockWith(ast.NewBreakStmt(p())))
	mustContain(t, d, "(break)")
}

func TestDumpStmtGoto(t *testing.T) {
	d := ast.DumpString(blockWith(ast.NewGotoStmt(p(), "end")))
	mustContain(t, d, "(goto end)")
}

func TestDumpStmtLabel(t *testing.T) {
	d := ast.DumpString(blockWith(ast.NewLabelStmt(p(), "start", 0)))
	mustContain(t, d, "(label start)")
}

func TestDumpStmtExprStmt(t *testing.T) {
	call := ast.NewFuncCallExpr(p(), ast.NewNameExpr(p(), "f"), nil)
	d := ast.DumpString(blockWith(ast.NewExprStmt(p(), call)))
	mustContain(t, d, "(expr-stmt")
	mustContain(t, d, "(call")
}

func TestDumpStmtFuncStmt(t *testing.T) {
	fn := ast.NewFuncExpr(p(),
		[]*ast.NameExpr{ast.NewNameExpr(p(), "x")},
		false, "", emptyBlock())
	s := ast.NewFuncStmt(p(), ast.NewNameExpr(p(), "foo"), false, fn)
	d := ast.DumpString(blockWith(s))
	mustContain(t, d, "(func-stmt")
	mustContain(t, d, "(name")
	mustContain(t, d, "(name foo)")
	mustContain(t, d, "(function (x)")
}

func TestDumpStmtFuncStmtMethod(t *testing.T) {
	fn := ast.NewFuncExpr(p(),
		[]*ast.NameExpr{ast.NewNameExpr(p(), "self")},
		false, "", emptyBlock())
	s := ast.NewFuncStmt(p(),
		ast.NewFieldExpr(p(), ast.NewNameExpr(p(), "obj"), "method"),
		true, fn)
	d := ast.DumpString(blockWith(s))
	mustContain(t, d, "(func-stmt :method")
	mustContain(t, d, "(field .method")
}

func TestDumpStmtLocalFunc(t *testing.T) {
	fn := ast.NewFuncExpr(p(), nil, false, "", emptyBlock())
	s := ast.NewLocalFuncStmt(p(), ast.NewNameExpr(p(), "helper"), fn)
	d := ast.DumpString(blockWith(s))
	mustContain(t, d, "(local-func helper")
}

func TestDumpStmtGlobal(t *testing.T) {
	s := ast.NewGlobalStmt(p(),
		[]*ast.NameExpr{ast.NewNameExpr(p(), "x")},
		[]string{""}, []ast.Expr{ast.NewNumberExpr(p(), 1, "1")})
	d := ast.DumpString(blockWith(s))
	mustContain(t, d, "(global")
	mustContain(t, d, "(names")
	mustContain(t, d, "x")
	mustContain(t, d, "(values")
}

func TestDumpStmtGlobalNoValues(t *testing.T) {
	s := ast.NewGlobalStmt(p(),
		[]*ast.NameExpr{ast.NewNameExpr(p(), "y")},
		[]string{""}, nil)
	d := ast.DumpString(blockWith(s))
	mustContain(t, d, "(global")
	mustNotContain(t, d, "(values")
}

func TestDumpStmtGlobalConst(t *testing.T) {
	s := ast.NewGlobalStmt(p(),
		[]*ast.NameExpr{ast.NewNameExpr(p(), "C")},
		[]string{"const"}, []ast.Expr{ast.NewNumberExpr(p(), 42, "42")})
	d := ast.DumpString(blockWith(s))
	mustContain(t, d, "C <const>")
}

func TestDumpStmtGlobalStar(t *testing.T) {
	s := ast.NewGlobalStarStmt(p(), "")
	d := ast.DumpString(blockWith(s))
	mustContain(t, d, "(global *)")
}

func TestDumpStmtGlobalFunc(t *testing.T) {
	fn := ast.NewFuncExpr(p(),
		[]*ast.NameExpr{ast.NewNameExpr(p(), "x")},
		false, "", emptyBlock())
	s := ast.NewGlobalFuncStmt(p(), ast.NewNameExpr(p(), "gfn"), fn)
	d := ast.DumpString(blockWith(s))
	mustContain(t, d, "(global-func gfn")
	mustContain(t, d, "(function (x)")
}

func TestDumpStmtEmpty(t *testing.T) {
	d := ast.DumpString(blockWith(ast.NewEmptyStmt(p())))
	mustContain(t, d, "(;)")
}

// ---------------------------------------------------------------------------
// Dump() writes to io.Writer
// ---------------------------------------------------------------------------

func TestDumpToWriter(t *testing.T) {
	var buf bytes.Buffer
	b := blockWith(ast.NewReturnStmt(p(), []ast.Expr{
		ast.NewNumberExpr(p(), 1, "1"),
	}))
	ast.Dump(&buf, b)
	got := buf.String()
	mustContain(t, got, "(return")
	mustContain(t, got, "(int 1)")
}

// ---------------------------------------------------------------------------
// NumberExpr / FloatExpr field verification
// ---------------------------------------------------------------------------

func TestNumberExprFields(t *testing.T) {
	n := ast.NewNumberExpr(pos(1, 5), 255, "0xFF")
	if n.Value != 255 {
		t.Errorf("Value = %d, want 255", n.Value)
	}
	if n.Raw != "0xFF" {
		t.Errorf("Raw = %q, want 0xFF", n.Raw)
	}
}

func TestFloatExprFields(t *testing.T) {
	f := ast.NewFloatExpr(pos(1, 1), 1e10, "1e10")
	if f.Value != 1e10 {
		t.Errorf("Value = %g, want 1e10", f.Value)
	}
	if f.Raw != "1e10" {
		t.Errorf("Raw = %q, want 1e10", f.Raw)
	}
}

func TestStringExprValue(t *testing.T) {
	s := ast.NewStringExpr(p(), "with\nnewline")
	if s.Value != "with\nnewline" {
		t.Errorf("Value = %q, want with\\nnewline", s.Value)
	}
}

// ---------------------------------------------------------------------------
// Parse → DumpString round-trip tests (exercises both ast.go and dump.go
// through real parser output)
// ---------------------------------------------------------------------------

func TestParseLiterals(t *testing.T) {
	tests := []struct {
		src  string
		want string
	}{
		{"return nil", "nil"},
		{"return true", "true"},
		{"return false", "false"},
		{"return 42", "(int 42)"},
		{"return 0xFF", "(int 0xFF)"},
		{"return 0b1010", "(int 0b1010)"},
		{"return 3.14", "(float 3.14)"},
		{"return 1e5", "(float 1e5)"},
		{"return .5", "(float .5)"},
		{"return 0x1p10", "(float 0x1p10)"},
		{`return "hello"`, `(string "hello")`},
		{`return 'world'`, `(string "world")`},
		{"return [[raw]]", `(string "raw")`},
	}
	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			d := parseDump(t, tt.src)
			mustContain(t, d, tt.want)
		})
	}
}

func TestParseBinaryOperators(t *testing.T) {
	ops := []string{"+", "-", "*", "/", "//", "%", "^", "..",
		"==", "~=", "<", "<=", ">", ">=",
		"and", "or", "&", "|", "~", "<<", ">>"}
	for _, op := range ops {
		t.Run(op, func(t *testing.T) {
			src := "return a " + op + " b"
			d := parseDump(t, src)
			mustContain(t, d, "(binop "+op)
		})
	}
}

func TestParseUnaryOperators(t *testing.T) {
	tests := []struct {
		src string
		op  string
	}{
		{"return -x", "-"},
		{"return not x", "not"},
		{"return #t", "#"},
		{"return ~n", "~"},
	}
	for _, tt := range tests {
		t.Run(tt.op, func(t *testing.T) {
			d := parseDump(t, tt.src)
			mustContain(t, d, "(unop "+tt.op)
		})
	}
}

func TestParsePrecedence(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []string
	}{
		{
			"add_mul", "return 1 + 2 * 3",
			[]string{"(binop +", "(binop *"},
		},
		{
			"unary_pow", "return -2 ^ 3",
			[]string{"(unop -", "(binop ^"},
		},
		{
			"pow_right_assoc", "return 2 ^ 3 ^ 4",
			[]string{"(binop ^"},
		},
		{
			"concat_right_assoc", "return 'a' .. 'b' .. 'c'",
			[]string{"(binop .."},
		},
		{
			"parens_override", "return (1 + 2) * 3",
			[]string{"(paren", "(binop +", "(binop *"},
		},
		{
			"and_or", "return a and b or c",
			[]string{"(binop or", "(binop and"},
		},
		{
			"compare_chain", "return a < b and b < c",
			[]string{"(binop and", "(binop <"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := parseDump(t, tt.src)
			for _, w := range tt.want {
				mustContain(t, d, w)
			}
		})
	}
}

func TestParseAssignment(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []string
	}{
		{"simple", "x = 1", []string{"(assign", "(name x)", "(int 1)"}},
		{"multi", "a, b = 1, 2", []string{"(name a)", "(name b)", "(int 1)", "(int 2)"}},
		{"field", "t.x = 1", []string{"(field .x"}},
		{"index", "t[1] = 2", []string{"(index"}},
		{"chained_field", "a.b.c = 1", []string{"(field .c", "(field .b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := parseDump(t, tt.src)
			for _, w := range tt.want {
				mustContain(t, d, w)
			}
		})
	}
}

func TestParseLocal(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []string
	}{
		{"simple", "local x = 1", []string{"(local", "x", "(int 1)"}},
		{"multi", "local a, b = 1, 2", []string{"a", "b"}},
		{"no_value", "local x", []string{"(local", "x"}},
		{"const", "local x <const> = 42", []string{"x <const>"}},
		{"close", "local f <close> = g()", []string{"f <close>"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := parseDump(t, tt.src)
			for _, w := range tt.want {
				mustContain(t, d, w)
			}
		})
	}
}

func TestParseDo(t *testing.T) {
	d := parseDump(t, "do local x = 1 end")
	mustContain(t, d, "(do")
	mustContain(t, d, "(local")
}

func TestParseWhile(t *testing.T) {
	d := parseDump(t, "while x > 0 do x = x - 1 end")
	mustContain(t, d, "(while")
	mustContain(t, d, "(binop >")
	mustContain(t, d, "(assign")
}

func TestParseRepeat(t *testing.T) {
	d := parseDump(t, "repeat x = x + 1 until x >= 10")
	mustContain(t, d, "(repeat")
	mustContain(t, d, "(until")
	mustContain(t, d, "(binop >=")
}

func TestParseIf(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []string
	}{
		{"simple", "if a then b() end", []string{"(if", "(then"}},
		{"else", "if a then b() else c() end", []string{"(if", "(then", "(else"}},
		{"elseif", "if a then b() elseif c then d() end", []string{"(if", "(elseif"}},
		{"full", "if a then b() elseif c then d() elseif e then f() else g() end",
			[]string{"(if", "(elseif", "(else"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := parseDump(t, tt.src)
			for _, w := range tt.want {
				mustContain(t, d, w)
			}
		})
	}
}

func TestParseForNum(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []string
	}{
		{"no_step", "for i = 1, 10 do print(i) end",
			[]string{"(for-num i", "(start", "(stop"}},
		{"with_step", "for i = 1, 100, 2 do print(i) end",
			[]string{"(for-num i", "(start", "(stop", "(step"}},
		{"negative_step", "for i = 10, 1, -1 do print(i) end",
			[]string{"(for-num i", "(step", "(unop -"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := parseDump(t, tt.src)
			for _, w := range tt.want {
				mustContain(t, d, w)
			}
		})
	}
}

func TestParseForIn(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []string
	}{
		{"pairs", "for k, v in pairs(t) do end",
			[]string{"(for-in [k, v]", "(iters"}},
		{"ipairs", "for i, v in ipairs(t) do end",
			[]string{"(for-in [i, v]"}},
		{"single_var", "for line in io.lines() do end",
			[]string{"(for-in [line]"}},
		{"next", "for k, v in next, t do end",
			[]string{"(for-in [k, v]"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := parseDump(t, tt.src)
			for _, w := range tt.want {
				mustContain(t, d, w)
			}
		})
	}
}

func TestParseReturn(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{"empty", "return", "(return)"},
		{"single", "return 1", "(int 1)"},
		{"multi", "return 1, 2, 3", "(int 3)"},
		{"expr", "return a + b", "(binop +"},
		{"call", "return f()", "(call"},
		{"semicolon", "return 1;", "(int 1)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := parseDump(t, tt.src)
			mustContain(t, d, tt.want)
		})
	}
}

func TestParseBreak(t *testing.T) {
	d := parseDump(t, "while true do break end")
	mustContain(t, d, "(break)")
}

func TestParseGotoLabel(t *testing.T) {
	d := parseDump(t, "goto done; ::done:: return")
	mustContain(t, d, "(goto done)")
	mustContain(t, d, "(label done)")
}

func TestParseFuncDecl(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []string
	}{
		{"simple", "function f() end",
			[]string{"(func-stmt", "(name f)"}},
		{"params", "function f(a, b, c) end",
			[]string{"(function (a, b, c)"}},
		{"vararg", "function f(...) end",
			[]string{"(function (...)"}},
		{"params_vararg", "function f(a, b, ...) end",
			[]string{"(function (a, b, ...)"}},
		{"dotted", "function a.b.c() end",
			[]string{"(func-stmt", "(field .c", "(field .b"}},
		{"method", "function obj:m() end",
			[]string{"(func-stmt :method", "(function (self)"}},
		{"method_params", "function obj:m(x, y) end",
			[]string{":method", "(function (self, x, y)"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := parseDump(t, tt.src)
			for _, w := range tt.want {
				mustContain(t, d, w)
			}
		})
	}
}

func TestParseLocalFunc(t *testing.T) {
	d := parseDump(t, "local function helper(x) return x * 2 end")
	mustContain(t, d, "(local-func helper")
	mustContain(t, d, "(function (x)")
	mustContain(t, d, "(binop *")
}

func TestParseFuncExpr(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []string
	}{
		{"anon", "local f = function() end",
			[]string{"(function ()"}},
		{"with_params", "local f = function(a, b) end",
			[]string{"(function (a, b)"}},
		{"vararg", "local f = function(...) end",
			[]string{"(function (...)"}},
		{"nested", "local f = function() return function() end end",
			[]string{"(function ()"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := parseDump(t, tt.src)
			for _, w := range tt.want {
				mustContain(t, d, w)
			}
		})
	}
}

func TestParseTableConstructor(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []string
	}{
		{"empty", "local t = {}", []string{"(table {})"}},
		{"list", "local t = {1, 2, 3}", []string{"(item", "(int 1)"}},
		{"record", `local t = {x = 1, y = 2}`, []string{`(string "x")`, `(string "y")`}},
		{"bracket_key", `local t = {[1+1] = "two"}`, []string{"(binop +"}},
		{"mixed", "local t = {1, x = 2, [3] = 4}",
			[]string{"(item", "(key", "(val"}},
		{"nested", "local t = {{1, 2}, {3, 4}}", []string{"(table {"}},
		{"trailing_comma", "local t = {1, 2, 3,}", []string{"(int 3)"}},
		{"trailing_semi", "local t = {1; 2; 3}", []string{"(int 3)"}},
		{"func_value", "local t = {f = function() end}", []string{"(function ()"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := parseDump(t, tt.src)
			for _, w := range tt.want {
				mustContain(t, d, w)
			}
		})
	}
}

func TestParseSuffixedExpr(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []string
	}{
		{"field", "return a.b", []string{"(field .b"}},
		{"chained_field", "return a.b.c.d", []string{"(field .d", "(field .c", "(field .b"}},
		{"index", "return a[1]", []string{"(index"}},
		{"chained_index", "return a[1][2]", []string{"(index"}},
		{"mixed_access", "return a.b[1].c", []string{"(field .c", "(index", "(field .b"}},
		{"call", "f(1, 2)", []string{"(call", "(args"}},
		{"call_no_args", "f()", []string{"(call", "(name f)"}},
		{"call_string", `require "foo"`, []string{"(call", `(string "foo")`}},
		{"call_table", "f{1, 2}", []string{"(call", "(table {"}},
		{"method_call", "obj:m(1)", []string{"(method-call :m"}},
		{"chained_calls", "a()()", []string{"(call"}},
		{"chained_method", "a:b():c()", []string{"(method-call :c", "(method-call :b"}},
		{"call_then_field", "return f().x", []string{"(field .x", "(call"}},
		{"paren_call", "(f)()", []string{"(paren", "(call"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := parseDump(t, tt.src)
			for _, w := range tt.want {
				mustContain(t, d, w)
			}
		})
	}
}

func TestParseParenExpr(t *testing.T) {
	d := parseDump(t, "return (a)")
	mustContain(t, d, "(paren")
	mustContain(t, d, "(name a)")
}

func TestParseVararg(t *testing.T) {
	d := parseDump(t, "local function f(...) return ... end")
	mustContain(t, d, "...")
}

func TestParseEmptyStmt(t *testing.T) {
	d := parseDump(t, ";;; return 1")
	mustContain(t, d, "(;)")
	mustContain(t, d, "(return")
}

// ---------------------------------------------------------------------------
// Complex programs (parse→dump)
// ---------------------------------------------------------------------------

func TestParseComplexFibonacci(t *testing.T) {
	src := `
local function fib(n)
    if n <= 1 then return n end
    return fib(n - 1) + fib(n - 2)
end
return fib(10)
`
	d := parseDump(t, src)
	mustContain(t, d, "(local-func fib")
	mustContain(t, d, "(if")
	mustContain(t, d, "(binop <=")
	mustContain(t, d, "(binop +")
	mustContain(t, d, "(call")
}

func TestParseComplexIterator(t *testing.T) {
	src := `
local function range(n)
    local i = 0
    return function()
        i = i + 1
        if i <= n then return i end
    end
end
for v in range(5) do print(v) end
`
	d := parseDump(t, src)
	mustContain(t, d, "(local-func range")
	mustContain(t, d, "(for-in [v]")
	mustContain(t, d, "(function ()")
}

func TestParseComplexClass(t *testing.T) {
	src := `
local Dog = {}
Dog.__index = Dog

function Dog.new(name)
    return setmetatable({name = name}, Dog)
end

function Dog:bark()
    return self.name .. " says woof!"
end
`
	d := parseDump(t, src)
	mustContain(t, d, "(func-stmt")
	mustContain(t, d, "(field .new")
	mustContain(t, d, ":method")
	mustContain(t, d, "(field .name")
	mustContain(t, d, "(binop ..")
}

func TestParseComplexNestedControl(t *testing.T) {
	src := `
for i = 1, 10 do
    if i % 2 == 0 then
        while i > 0 do
            i = i - 1
            if i == 3 then goto skip end
        end
        ::skip::
    elseif i == 5 then
        repeat
            i = i + 1
        until i > 7
    else
        break
    end
end
`
	d := parseDump(t, src)
	mustContain(t, d, "(for-num i")
	mustContain(t, d, "(if")
	mustContain(t, d, "(while")
	mustContain(t, d, "(goto skip)")
	mustContain(t, d, "(label skip)")
	mustContain(t, d, "(elseif")
	mustContain(t, d, "(repeat")
	mustContain(t, d, "(until")
	mustContain(t, d, "(else")
	mustContain(t, d, "(break)")
}

func TestParseComplexTablePatterns(t *testing.T) {
	src := `
local config = {
    name = "test",
    values = {1, 2, 3},
    nested = {
        deep = {
            [true] = "yes",
            [false] = "no",
        }
    },
    callback = function(x, y)
        return x + y
    end,
    [1] = "first",
    ["key with spaces"] = 42,
}
`
	d := parseDump(t, src)
	mustContain(t, d, `(string "name")`)
	mustContain(t, d, `(string "values")`)
	mustContain(t, d, `(string "nested")`)
	mustContain(t, d, `(string "deep")`)
	mustContain(t, d, "true")
	mustContain(t, d, "false")
	mustContain(t, d, "(function (x, y)")
	mustContain(t, d, `(string "key with spaces")`)
}

func TestParseComplexClosures(t *testing.T) {
	src := `
local function make_counter()
    local count = 0
    return {
        inc = function() count = count + 1 end,
        get = function() return count end,
    }
end
`
	d := parseDump(t, src)
	mustContain(t, d, "(local-func make_counter")
	mustContain(t, d, `(string "inc")`)
	mustContain(t, d, `(string "get")`)
	mustContain(t, d, "(function ()")
}

func TestParseComplexMultiReturn(t *testing.T) {
	src := `
local function swap(a, b)
    return b, a
end
local x, y = swap(1, 2)
`
	d := parseDump(t, src)
	mustContain(t, d, "(local-func swap")
	mustContain(t, d, "(return")
	mustContain(t, d, "(name b)")
	mustContain(t, d, "(name a)")
}

func TestParseComplexDoBlock(t *testing.T) {
	src := `
do
    local x = 10
    do
        local y = 20
        return x + y
    end
end
`
	d := parseDump(t, src)
	mustContain(t, d, "(do")
	mustContain(t, d, "(local")
	mustContain(t, d, "(binop +")
}

func TestParseComplexChainedMethodCalls(t *testing.T) {
	src := `
return builder:new()
    :set("name", "test")
    :set("value", 42)
    :build()
`
	d := parseDump(t, src)
	mustContain(t, d, "(method-call :build")
	mustContain(t, d, "(method-call :set")
	mustContain(t, d, "(method-call :new")
}

func TestParseComplexVarargForward(t *testing.T) {
	src := `
local function log(level, ...)
    print(level, ...)
end
`
	d := parseDump(t, src)
	mustContain(t, d, "(local-func log")
	mustContain(t, d, "(function (level, ...)")
	mustContain(t, d, "...")
}

func TestParseComplexGotoPattern(t *testing.T) {
	src := `
for i = 1, 100 do
    if i % 15 == 0 then goto fizzbuzz end
    if i % 3 == 0 then goto fizz end
    if i % 5 == 0 then goto buzz end
    goto continue
    ::fizzbuzz:: print("fizzbuzz") goto continue
    ::fizz:: print("fizz") goto continue
    ::buzz:: print("buzz") goto continue
    ::continue::
end
`
	d := parseDump(t, src)
	mustContain(t, d, "(goto fizzbuzz)")
	mustContain(t, d, "(goto fizz)")
	mustContain(t, d, "(goto buzz)")
	mustContain(t, d, "(goto continue)")
	mustContain(t, d, "(label fizzbuzz)")
	mustContain(t, d, "(label fizz)")
	mustContain(t, d, "(label buzz)")
	mustContain(t, d, "(label continue)")
}

func TestParseComplexStringCalls(t *testing.T) {
	// Different call syntaxes
	src := `
print "hello"
f {1, 2, 3}
t:method "arg"
t:method {x = 1}
`
	d := parseDump(t, src)
	mustContain(t, d, `(string "hello")`)
	mustContain(t, d, "(table {")
	mustContain(t, d, "(method-call :method")
}

func TestParseComplexAssignmentTargets(t *testing.T) {
	src := `
a, b, c = 1, 2, 3
t.x, t.y = t.y, t.x
a[1], a[2] = a[2], a[1]
`
	d := parseDump(t, src)
	mustContain(t, d, "(assign")
	mustContain(t, d, "(field .x")
	mustContain(t, d, "(field .y")
	mustContain(t, d, "(index")
}

func TestParseComplexNestedFunctions(t *testing.T) {
	src := `
function outer()
    local function middle()
        local function inner()
            return 42
        end
        return inner
    end
    return middle
end
`
	d := parseDump(t, src)
	mustContain(t, d, "(func-stmt")
	mustContain(t, d, "(local-func middle")
	mustContain(t, d, "(local-func inner")
	mustContain(t, d, "(int 42)")
}

func TestParseComplexBitwiseOps(t *testing.T) {
	src := "return (a & b) | (c ~ d), a << 2, b >> 3"
	d := parseDump(t, src)
	mustContain(t, d, "(binop &")
	mustContain(t, d, "(binop |")
	mustContain(t, d, "(binop ~")
	mustContain(t, d, "(binop <<")
	mustContain(t, d, "(binop >>")
}

func TestParseComplexIntegerDivision(t *testing.T) {
	d := parseDump(t, "return a // b + c % d")
	mustContain(t, d, "(binop //")
	mustContain(t, d, "(binop %")
}

func TestParseComplexExpressionStatements(t *testing.T) {
	src := `
print(1)
t:method()
(getfunc())(arg)
`
	d := parseDump(t, src)
	// Each of these should produce an expr-stmt wrapping a call
	mustContain(t, d, "(expr-stmt")
	mustContain(t, d, "(call")
	mustContain(t, d, "(method-call :method")
}

func TestParseComplexMultipleLocals(t *testing.T) {
	src := `
local a, b, c
local x = 1
local y, z = 2, 3
local pi <const> = 3.14
`
	d := parseDump(t, src)
	// 4 local statements
	count := strings.Count(d, "(local")
	if count < 4 {
		t.Errorf("expected at least 4 (local blocks, got %d in:\n%s", count, d)
	}
	mustContain(t, d, "pi <const>")
}

func TestParseComplexForInMultiIter(t *testing.T) {
	// for-in with multiple iterator expressions (e.g., next, t, nil)
	d := parseDump(t, "for k, v in next, t, nil do end")
	mustContain(t, d, "(for-in [k, v]")
	mustContain(t, d, "(iters")
	mustContain(t, d, "(name next)")
	mustContain(t, d, "(name t)")
	mustContain(t, d, "nil")
}

func TestParseComplexWhileBreak(t *testing.T) {
	src := `
while true do
    local x = f()
    if x == nil then break end
    process(x)
end
`
	d := parseDump(t, src)
	mustContain(t, d, "(while")
	mustContain(t, d, "true")
	mustContain(t, d, "(break)")
	mustContain(t, d, "(if")
}

func TestParseComplexRepeatUntil(t *testing.T) {
	src := `
local x = 0
repeat
    x = x + 1
    local y = x * x
until y > 100
`
	d := parseDump(t, src)
	mustContain(t, d, "(repeat")
	mustContain(t, d, "(until")
	mustContain(t, d, "(binop >")
}

func TestParseComplexDeeplyNestedExpr(t *testing.T) {
	d := parseDump(t, "return ((((1 + 2))))")
	mustContain(t, d, "(paren")
	mustContain(t, d, "(binop +")
}

func TestParseComplexEmptyFunction(t *testing.T) {
	d := parseDump(t, "local function noop() end")
	mustContain(t, d, "(local-func noop")
	mustContain(t, d, "(block)")
}

func TestParseComplexImmediateFuncCall(t *testing.T) {
	d := parseDump(t, "(function() print('hello') end)()")
	mustContain(t, d, "(call")
	mustContain(t, d, "(paren")
	mustContain(t, d, "(function ()")
}

func TestParseComplexStringEscapes(t *testing.T) {
	d := parseDump(t, `return "\n\t\\\"\'"`)
	mustContain(t, d, "(string")
}

func TestParseComplexLongString(t *testing.T) {
	d := parseDump(t, "return [==[hello]==]")
	mustContain(t, d, `(string "hello")`)
}

func TestParseComplexSemicolonsEverywhere(t *testing.T) {
	src := ";local x = 1; x = 2; return x;"
	d := parseDump(t, src)
	mustContain(t, d, "(;)")
	mustContain(t, d, "(local")
	mustContain(t, d, "(assign")
	mustContain(t, d, "(return")
}

func TestParseComplexEmptyTable(t *testing.T) {
	d := parseDump(t, "return {}, {}, {}")
	count := strings.Count(d, "(table {})")
	if count != 3 {
		t.Errorf("expected 3 empty tables, got %d in:\n%s", count, d)
	}
}

func TestParseComplexNestedMethodCall(t *testing.T) {
	d := parseDump(t, "a.b.c:d(e.f.g)")
	mustContain(t, d, "(method-call :d")
	mustContain(t, d, "(field .c")
	mustContain(t, d, "(field .b")
	mustContain(t, d, "(field .g")
	mustContain(t, d, "(field .f")
}

func TestParseComplexMultipleReturns(t *testing.T) {
	src := `
if true then return 1 end
return 2
`
	d := parseDump(t, src)
	mustContain(t, d, "(int 1)")
	mustContain(t, d, "(int 2)")
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestParseMinimalProgram(t *testing.T) {
	d := parseDump(t, "")
	mustContain(t, d, "(block)")
}

func TestParseSingleSemicolon(t *testing.T) {
	d := parseDump(t, ";")
	mustContain(t, d, "(;)")
}

func TestParseOnlyReturn(t *testing.T) {
	d := parseDump(t, "return")
	mustContain(t, d, "(return)")
}

func TestParseHexNumbers(t *testing.T) {
	d := parseDump(t, "return 0xFF, 0XAB")
	mustContain(t, d, "(int 0xFF)")
	mustContain(t, d, "(int 0XAB)")
}

func TestParseNegativeNumbers(t *testing.T) {
	// Negative numbers are parsed as unop(-number)
	d := parseDump(t, "return -1, -3.14")
	mustContain(t, d, "(unop -")
	mustContain(t, d, "(int 1)")
	mustContain(t, d, "(float 3.14)")
}

func TestParseStringWithEscapedNewline(t *testing.T) {
	d := parseDump(t, `return "line1\nline2"`)
	mustContain(t, d, "(string")
}

func TestParseFuncCallAsStatement(t *testing.T) {
	d := parseDump(t, "print(1)")
	mustContain(t, d, "(expr-stmt")
	mustContain(t, d, "(call")
}

func TestParseMethodCallAsStatement(t *testing.T) {
	d := parseDump(t, "obj:method()")
	mustContain(t, d, "(expr-stmt")
	mustContain(t, d, "(method-call :method")
}

func TestDumpFuncExprNoParams(t *testing.T) {
	fe := ast.NewFuncExpr(p(), nil, false, "", emptyBlock())
	b := blockWith(ast.NewReturnStmt(p(), []ast.Expr{fe}))
	d := ast.DumpString(b)
	mustContain(t, d, "(function ()")
}

func TestDumpMultipleStmts(t *testing.T) {
	b := blockWith(
		ast.NewLocalStmt(p(),
			[]*ast.NameExpr{ast.NewNameExpr(p(), "x")},
			[]string{""}, []ast.Expr{ast.NewNumberExpr(p(), 1, "1")}),
		ast.NewAssignStmt(p(),
			[]ast.Expr{ast.NewNameExpr(p(), "x")},
			[]ast.Expr{ast.NewBinopExpr(p(), "+",
				ast.NewNameExpr(p(), "x"),
				ast.NewNumberExpr(p(), 1, "1"))}),
		ast.NewReturnStmt(p(), []ast.Expr{ast.NewNameExpr(p(), "x")}),
	)
	d := ast.DumpString(b)
	mustContain(t, d, "(local")
	mustContain(t, d, "(assign")
	mustContain(t, d, "(return")
	mustContain(t, d, "(binop +")
}

// ---------------------------------------------------------------------------
// Field value verification
// ---------------------------------------------------------------------------

func TestBinopExprFields(t *testing.T) {
	left := ast.NewNumberExpr(p(), 1, "1")
	right := ast.NewNumberExpr(p(), 2, "2")
	b := ast.NewBinopExpr(pos(3, 5), "+", left, right)
	if b.Op != "+" {
		t.Errorf("Op = %q, want +", b.Op)
	}
	if b.Left != left {
		t.Error("Left mismatch")
	}
	if b.Right != right {
		t.Error("Right mismatch")
	}
}

func TestUnopExprFields(t *testing.T) {
	operand := ast.NewNameExpr(p(), "x")
	u := ast.NewUnopExpr(pos(1, 1), "not", operand)
	if u.Op != "not" {
		t.Errorf("Op = %q, want not", u.Op)
	}
	if u.Operand != operand {
		t.Error("Operand mismatch")
	}
}

func TestIndexExprFields(t *testing.T) {
	tbl := ast.NewNameExpr(p(), "t")
	key := ast.NewStringExpr(p(), "k")
	ie := ast.NewIndexExpr(pos(2, 3), tbl, key)
	if ie.Table != tbl {
		t.Error("Table mismatch")
	}
	if ie.Key != key {
		t.Error("Key mismatch")
	}
}

func TestFieldExprFields(t *testing.T) {
	tbl := ast.NewNameExpr(p(), "obj")
	fe := ast.NewFieldExpr(pos(1, 1), tbl, "name")
	if fe.Table != tbl {
		t.Error("Table mismatch")
	}
	if fe.Field != "name" {
		t.Errorf("Field = %q, want name", fe.Field)
	}
}

func TestMethodCallExprFields(t *testing.T) {
	obj := ast.NewNameExpr(p(), "o")
	args := []ast.Expr{ast.NewNumberExpr(p(), 1, "1")}
	mc := ast.NewMethodCallExpr(pos(1, 1), obj, "do_thing", args)
	if mc.Object != obj {
		t.Error("Object mismatch")
	}
	if mc.Method != "do_thing" {
		t.Errorf("Method = %q, want do_thing", mc.Method)
	}
	if len(mc.Args) != 1 {
		t.Errorf("Args len = %d, want 1", len(mc.Args))
	}
}

func TestFuncCallExprFields(t *testing.T) {
	fn := ast.NewNameExpr(p(), "f")
	args := []ast.Expr{ast.NewTrueExpr(p()), ast.NewFalseExpr(p())}
	fc := ast.NewFuncCallExpr(pos(1, 1), fn, args)
	if fc.Func != fn {
		t.Error("Func mismatch")
	}
	if len(fc.Args) != 2 {
		t.Errorf("Args len = %d, want 2", len(fc.Args))
	}
}

func TestFuncExprFields(t *testing.T) {
	params := []*ast.NameExpr{ast.NewNameExpr(p(), "a"), ast.NewNameExpr(p(), "b")}
	body := emptyBlock()
	fe := ast.NewFuncExpr(pos(1, 1), params, true, "rest", body)
	if len(fe.Params) != 2 {
		t.Errorf("Params len = %d, want 2", len(fe.Params))
	}
	if !fe.VarArg {
		t.Error("VarArg should be true")
	}
	if fe.VarArgName != "rest" {
		t.Errorf("VarArgName = %q, want rest", fe.VarArgName)
	}
	if fe.Body != body {
		t.Error("Body mismatch")
	}
}

func TestAssignStmtFields(t *testing.T) {
	targets := []ast.Expr{ast.NewNameExpr(p(), "x")}
	values := []ast.Expr{ast.NewNumberExpr(p(), 1, "1")}
	s := ast.NewAssignStmt(pos(1, 1), targets, values)
	if len(s.Targets) != 1 {
		t.Errorf("Targets len = %d, want 1", len(s.Targets))
	}
	if len(s.Values) != 1 {
		t.Errorf("Values len = %d, want 1", len(s.Values))
	}
}

func TestLocalStmtFields(t *testing.T) {
	names := []*ast.NameExpr{ast.NewNameExpr(p(), "a")}
	attribs := []string{"const"}
	vals := []ast.Expr{ast.NewNumberExpr(p(), 42, "42")}
	s := ast.NewLocalStmt(pos(1, 1), names, attribs, vals)
	if len(s.Names) != 1 || s.Names[0].Name != "a" {
		t.Error("Names mismatch")
	}
	if len(s.Attribs) != 1 || s.Attribs[0] != "const" {
		t.Error("Attribs mismatch")
	}
}

func TestIfStmtFields(t *testing.T) {
	cond := ast.NewTrueExpr(p())
	thenBlock := emptyBlock()
	elseBlock := emptyBlock()
	elseifs := []*ast.ElseIf{ast.NewElseIf(p(), ast.NewFalseExpr(p()), 0, emptyBlock())}
	s := ast.NewIfStmt(pos(1, 1), cond, 0, thenBlock, elseifs, elseBlock, 0)
	if s.Cond != cond {
		t.Error("Cond mismatch")
	}
	if s.Then != thenBlock {
		t.Error("Then mismatch")
	}
	if len(s.ElseIfs) != 1 {
		t.Errorf("ElseIfs len = %d, want 1", len(s.ElseIfs))
	}
	if s.Else != elseBlock {
		t.Error("Else mismatch")
	}
}

func TestForNumStmtFields(t *testing.T) {
	name := ast.NewNameExpr(p(), "i")
	start := ast.NewNumberExpr(p(), 1, "1")
	stop := ast.NewNumberExpr(p(), 10, "10")
	step := ast.NewNumberExpr(p(), 2, "2")
	body := emptyBlock()
	s := ast.NewForNumStmt(pos(1, 1), name, start, stop, step, body, 0)
	if s.Name != name {
		t.Error("Name mismatch")
	}
	if s.Start != start {
		t.Error("Start mismatch")
	}
	if s.Stop != stop {
		t.Error("Stop mismatch")
	}
	if s.Step != step {
		t.Error("Step mismatch")
	}
	if s.Body != body {
		t.Error("Body mismatch")
	}
}

func TestForInStmtFields(t *testing.T) {
	names := []*ast.NameExpr{ast.NewNameExpr(p(), "k"), ast.NewNameExpr(p(), "v")}
	iters := []ast.Expr{ast.NewNameExpr(p(), "pairs")}
	body := emptyBlock()
	s := ast.NewForInStmt(pos(1, 1), names, iters, body, 0)
	if len(s.Names) != 2 {
		t.Errorf("Names len = %d, want 2", len(s.Names))
	}
	if len(s.Iters) != 1 {
		t.Errorf("Iters len = %d, want 1", len(s.Iters))
	}
}

func TestGotoStmtFields(t *testing.T) {
	s := ast.NewGotoStmt(pos(1, 1), "target")
	if s.Label != "target" {
		t.Errorf("Label = %q, want target", s.Label)
	}
}

func TestLabelStmtFields(t *testing.T) {
	s := ast.NewLabelStmt(pos(1, 1), "target", 1)
	if s.Name != "target" {
		t.Errorf("Name = %q, want target", s.Name)
	}
}

func TestFuncStmtFields(t *testing.T) {
	name := ast.NewNameExpr(p(), "f")
	fn := ast.NewFuncExpr(p(), nil, false, "", emptyBlock())
	s := ast.NewFuncStmt(pos(1, 1), name, true, fn)
	if s.Name != name {
		t.Error("Name mismatch")
	}
	if !s.IsMethod {
		t.Error("IsMethod should be true")
	}
	if s.Func != fn {
		t.Error("Func mismatch")
	}
}

func TestGlobalStmtFields(t *testing.T) {
	s := ast.NewGlobalStmt(pos(1, 1),
		[]*ast.NameExpr{ast.NewNameExpr(p(), "x")},
		[]string{"const"},
		[]ast.Expr{ast.NewNumberExpr(p(), 1, "1")})
	if len(s.Names) != 1 {
		t.Errorf("Names len = %d, want 1", len(s.Names))
	}
	if s.Star {
		t.Error("Star should be false")
	}
}

func TestGlobalFuncStmtFields(t *testing.T) {
	name := ast.NewNameExpr(p(), "f")
	fn := ast.NewFuncExpr(p(), nil, false, "", emptyBlock())
	s := ast.NewGlobalFuncStmt(pos(1, 1), name, fn)
	if s.Name != name {
		t.Error("Name mismatch")
	}
	if s.Func != fn {
		t.Error("Func mismatch")
	}
}

func TestParenExprFields(t *testing.T) {
	inner := ast.NewNumberExpr(p(), 1, "1")
	pe := ast.NewParenExpr(pos(1, 1), inner)
	if pe.Inner != inner {
		t.Error("Inner mismatch")
	}
}

func TestTableConstructorFields(t *testing.T) {
	fields := []*ast.TableField{
		ast.NewTableField(p(), nil, ast.NewNumberExpr(p(), 1, "1")),
		ast.NewTableField(p(), ast.NewStringExpr(p(), "k"), ast.NewNumberExpr(p(), 2, "2")),
	}
	tc := ast.NewTableConstructor(pos(1, 1), fields)
	if len(tc.Fields) != 2 {
		t.Errorf("Fields len = %d, want 2", len(tc.Fields))
	}
	if tc.Fields[0].Key != nil {
		t.Error("First field key should be nil (positional)")
	}
	if tc.Fields[1].Key == nil {
		t.Error("Second field key should not be nil")
	}
}
