package ast

import (
	"fmt"
	"io"
	"strings"
)

// Dump writes a human-readable, indented representation of the AST to w.
func Dump(w io.Writer, block *Block) {
	d := &dumper{w: w}
	d.block(block)
}

// DumpString returns a human-readable string representation of the AST.
func DumpString(block *Block) string {
	var buf strings.Builder
	Dump(&buf, block)
	return buf.String()
}

type dumper struct {
	w     io.Writer
	depth int
}

func (d *dumper) indent() {
	for i := 0; i < d.depth; i++ {
		fmt.Fprint(d.w, "  ")
	}
}

func (d *dumper) printf(format string, args ...any) {
	d.indent()
	fmt.Fprintf(d.w, format, args...)
	fmt.Fprintln(d.w)
}

func (d *dumper) block(b *Block) {
	if b == nil || len(b.Stmts) == 0 {
		d.printf("(block)")
		return
	}
	d.printf("(block")
	d.depth++
	for _, s := range b.Stmts {
		d.stmt(s)
	}
	d.depth--
	d.printf(")")
}

func (d *dumper) stmt(s Stmt) {
	switch s := s.(type) {
	case *AssignStmt:
		d.printf("(assign")
		d.depth++
		d.printf("(targets")
		d.depth++
		for _, t := range s.Targets {
			d.expr(t)
		}
		d.depth--
		d.printf(")")
		d.printf("(values")
		d.depth++
		for _, v := range s.Values {
			d.expr(v)
		}
		d.depth--
		d.printf(")")
		d.depth--
		d.printf(")")

	case *LocalStmt:
		d.printf("(local")
		d.depth++
		d.printf("(names")
		d.depth++
		for i, n := range s.Names {
			attr := ""
			if i < len(s.Attribs) && s.Attribs[i] != "" {
				attr = fmt.Sprintf(" <%s>", s.Attribs[i])
			}
			d.printf("%s%s", n.Name, attr)
		}
		d.depth--
		d.printf(")")
		if len(s.Values) > 0 {
			d.printf("(values")
			d.depth++
			for _, v := range s.Values {
				d.expr(v)
			}
			d.depth--
			d.printf(")")
		}
		d.depth--
		d.printf(")")

	case *DoStmt:
		d.printf("(do")
		d.depth++
		d.block(s.Body)
		d.depth--
		d.printf(")")

	case *WhileStmt:
		d.printf("(while")
		d.depth++
		d.expr(s.Cond)
		d.block(s.Body)
		d.depth--
		d.printf(")")

	case *RepeatStmt:
		d.printf("(repeat")
		d.depth++
		d.block(s.Body)
		d.printf("(until")
		d.depth++
		d.expr(s.Cond)
		d.depth--
		d.printf(")")
		d.depth--
		d.printf(")")

	case *IfStmt:
		d.printf("(if")
		d.depth++
		d.expr(s.Cond)
		d.printf("(then")
		d.depth++
		d.block(s.Then)
		d.depth--
		d.printf(")")
		for _, elif := range s.ElseIfs {
			d.printf("(elseif")
			d.depth++
			d.expr(elif.Cond)
			d.printf("(then")
			d.depth++
			d.block(elif.Then)
			d.depth--
			d.printf(")")
			d.depth--
			d.printf(")")
		}
		if s.Else != nil {
			d.printf("(else")
			d.depth++
			d.block(s.Else)
			d.depth--
			d.printf(")")
		}
		d.depth--
		d.printf(")")

	case *ForNumStmt:
		d.printf("(for-num %s", s.Name.Name)
		d.depth++
		d.printf("(start")
		d.depth++
		d.expr(s.Start)
		d.depth--
		d.printf(")")
		d.printf("(stop")
		d.depth++
		d.expr(s.Stop)
		d.depth--
		d.printf(")")
		if s.Step != nil {
			d.printf("(step")
			d.depth++
			d.expr(s.Step)
			d.depth--
			d.printf(")")
		}
		d.block(s.Body)
		d.depth--
		d.printf(")")

	case *ForInStmt:
		names := make([]string, len(s.Names))
		for i, n := range s.Names {
			names[i] = n.Name
		}
		d.printf("(for-in [%s]", strings.Join(names, ", "))
		d.depth++
		d.printf("(iters")
		d.depth++
		for _, it := range s.Iters {
			d.expr(it)
		}
		d.depth--
		d.printf(")")
		d.block(s.Body)
		d.depth--
		d.printf(")")

	case *ReturnStmt:
		if len(s.Values) == 0 {
			d.printf("(return)")
		} else {
			d.printf("(return")
			d.depth++
			for _, v := range s.Values {
				d.expr(v)
			}
			d.depth--
			d.printf(")")
		}

	case *BreakStmt:
		d.printf("(break)")

	case *GotoStmt:
		d.printf("(goto %s)", s.Label)

	case *LabelStmt:
		d.printf("(label %s)", s.Name)

	case *ExprStmt:
		d.printf("(expr-stmt")
		d.depth++
		d.expr(s.Expr)
		d.depth--
		d.printf(")")

	case *FuncStmt:
		method := ""
		if s.IsMethod {
			method = " :method"
		}
		d.printf("(func-stmt%s", method)
		d.depth++
		d.printf("(name")
		d.depth++
		d.expr(s.Name)
		d.depth--
		d.printf(")")
		d.funcExpr(s.Func)
		d.depth--
		d.printf(")")

	case *LocalFuncStmt:
		d.printf("(local-func %s", s.Name.Name)
		d.depth++
		d.funcExpr(s.Func)
		d.depth--
		d.printf(")")

	case *GlobalStmt:
		if s.Star {
			d.printf("(global *)")
		} else {
			d.printf("(global")
			d.depth++
			d.printf("(names")
			d.depth++
			for i, n := range s.Names {
				attr := ""
				if i < len(s.Attribs) && s.Attribs[i] != "" {
					attr = fmt.Sprintf(" <%s>", s.Attribs[i])
				}
				d.printf("%s%s", n.Name, attr)
			}
			d.depth--
			d.printf(")")
			if len(s.Values) > 0 {
				d.printf("(values")
				d.depth++
				for _, v := range s.Values {
					d.expr(v)
				}
				d.depth--
				d.printf(")")
			}
			d.depth--
			d.printf(")")
		}

	case *GlobalFuncStmt:
		d.printf("(global-func %s", s.Name.Name)
		d.depth++
		d.funcExpr(s.Func)
		d.depth--
		d.printf(")")

	case *EmptyStmt:
		d.printf("(;)")

	default:
		d.printf("(?unknown-stmt)")
	}
}

func (d *dumper) expr(e Expr) {
	switch e := e.(type) {
	case *NilExpr:
		d.printf("nil")
	case *TrueExpr:
		d.printf("true")
	case *FalseExpr:
		d.printf("false")
	case *NumberExpr:
		d.printf("(int %s)", e.Raw)
	case *FloatExpr:
		d.printf("(float %s)", e.Raw)
	case *StringExpr:
		d.printf("(string %q)", e.Value)
	case *VarArgExpr:
		d.printf("...")
	case *NameExpr:
		d.printf("(name %s)", e.Name)

	case *BinopExpr:
		d.printf("(binop %s", e.Op)
		d.depth++
		d.expr(e.Left)
		d.expr(e.Right)
		d.depth--
		d.printf(")")

	case *UnopExpr:
		d.printf("(unop %s", e.Op)
		d.depth++
		d.expr(e.Operand)
		d.depth--
		d.printf(")")

	case *IndexExpr:
		d.printf("(index")
		d.depth++
		d.expr(e.Table)
		d.expr(e.Key)
		d.depth--
		d.printf(")")

	case *FieldExpr:
		d.printf("(field .%s", e.Field)
		d.depth++
		d.expr(e.Table)
		d.depth--
		d.printf(")")

	case *MethodCallExpr:
		d.printf("(method-call :%s", e.Method)
		d.depth++
		d.expr(e.Object)
		if len(e.Args) > 0 {
			d.printf("(args")
			d.depth++
			for _, a := range e.Args {
				d.expr(a)
			}
			d.depth--
			d.printf(")")
		}
		d.depth--
		d.printf(")")

	case *FuncCallExpr:
		d.printf("(call")
		d.depth++
		d.expr(e.Func)
		if len(e.Args) > 0 {
			d.printf("(args")
			d.depth++
			for _, a := range e.Args {
				d.expr(a)
			}
			d.depth--
			d.printf(")")
		}
		d.depth--
		d.printf(")")

	case *FuncExpr:
		d.funcExpr(e)

	case *TableConstructor:
		if len(e.Fields) == 0 {
			d.printf("(table {})")
		} else {
			d.printf("(table {")
			d.depth++
			for _, f := range e.Fields {
				if f.Key != nil {
					d.printf("(field")
					d.depth++
					d.printf("(key")
					d.depth++
					d.expr(f.Key)
					d.depth--
					d.printf(")")
					d.printf("(val")
					d.depth++
					d.expr(f.Value)
					d.depth--
					d.printf(")")
					d.depth--
					d.printf(")")
				} else {
					d.printf("(item")
					d.depth++
					d.expr(f.Value)
					d.depth--
					d.printf(")")
				}
			}
			d.depth--
			d.printf("})")
		}

	case *ParenExpr:
		d.printf("(paren")
		d.depth++
		d.expr(e.Inner)
		d.depth--
		d.printf(")")

	default:
		d.printf("(?unknown-expr)")
	}
}

func (d *dumper) funcExpr(f *FuncExpr) {
	params := make([]string, len(f.Params))
	for i, p := range f.Params {
		params[i] = p.Name
	}
	va := ""
	if f.VarArg {
		if f.VarArgName != "" {
			va = fmt.Sprintf(", ...%s", f.VarArgName)
		} else {
			va = ", ..."
		}
	}
	if len(params) == 0 && f.VarArg {
		va = strings.TrimPrefix(va, ", ")
	}
	d.printf("(function (%s%s)", strings.Join(params, ", "), va)
	d.depth++
	d.block(f.Body)
	d.depth--
	d.printf(")")
}
