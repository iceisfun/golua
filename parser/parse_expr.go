package parser

import (
	"github.com/iceisfun/golua/ast"
	"github.com/iceisfun/golua/token"
)

// ---------------------------------------------------------------------------
// Expression parsing (precedence climbing)
// ---------------------------------------------------------------------------

type binPriority struct{ left, right int }

var priorities = map[string]binPriority{
	"+":   {10, 10},
	"-":   {10, 10},
	"*":   {11, 11},
	"%":   {11, 11},
	"^":   {14, 13}, // right-assoc
	"/":   {11, 11},
	"//":  {11, 11},
	"&":   {6, 6},
	"|":   {4, 4},
	"~":   {5, 5},
	"<<":  {7, 7},
	">>":  {7, 7},
	"..":  {9, 8}, // right-assoc
	"==":  {3, 3},
	"<":   {3, 3},
	"<=":  {3, 3},
	"~=":  {3, 3},
	">":   {3, 3},
	">=":  {3, 3},
	"and": {2, 2},
	"or":  {1, 1},
}

const unaryPriority = 12

func (p *parser) getBinop() string {
	switch p.tok.Type {
	case token.Type('+'):
		return "+"
	case token.Type('-'):
		return "-"
	case token.Type('*'):
		return "*"
	case token.Type('/'):
		return "/"
	case token.IDIV:
		return "//"
	case token.Type('%'):
		return "%"
	case token.Type('^'):
		return "^"
	case token.Type('&'):
		return "&"
	case token.Type('|'):
		return "|"
	case token.Type('~'):
		return "~"
	case token.SHL:
		return "<<"
	case token.SHR:
		return ">>"
	case token.CONCAT:
		return ".."
	case token.EQ:
		return "=="
	case token.NE:
		return "~="
	case token.Type('<'):
		return "<"
	case token.LE:
		return "<="
	case token.Type('>'):
		return ">"
	case token.GE:
		return ">="
	case token.AND:
		return "and"
	case token.OR:
		return "or"
	default:
		return ""
	}
}

func (p *parser) getUnop() string {
	switch p.tok.Type {
	case token.NOT:
		return "not"
	case token.Type('-'):
		return "-"
	case token.Type('~'):
		return "~"
	case token.Type('#'):
		return "#"
	default:
		return ""
	}
}

func (p *parser) parseExpr() ast.Expr {
	return p.parseSubExpr(0)
}

// parseSubExpr -> (simpleexp | unop subexpr) { binop subexpr }
func (p *parser) parseSubExpr(limit int) ast.Expr {
	if p.err != nil {
		return ast.NewNilExpr(p.pos())
	}

	var expr ast.Expr

	if uop := p.getUnop(); uop != "" {
		pos := p.pos()
		p.advance()
		operand := p.parseSubExpr(unaryPriority)
		expr = ast.NewUnopExpr(pos, uop, operand)
	} else {
		expr = p.parseSimpleExpr()
	}

	for {
		op := p.getBinop()
		if op == "" {
			break
		}
		pri := priorities[op]
		if pri.left <= limit {
			break
		}
		pos := p.pos()
		p.advance()
		right := p.parseSubExpr(pri.right)
		expr = ast.NewBinopExpr(pos, op, expr, right)
	}

	return expr
}

func (p *parser) parseSimpleExpr() ast.Expr {
	if p.err != nil {
		return ast.NewNilExpr(p.pos())
	}
	pos := p.pos()
	switch p.tok.Type {
	case token.INT:
		v := p.tok
		p.advance()
		return ast.NewNumberExpr(pos, v.IntVal, v.Literal)
	case token.FLOAT:
		v := p.tok
		p.advance()
		return ast.NewFloatExpr(pos, v.FltVal, v.Literal)
	case token.STRING:
		v := p.tok
		p.advance()
		return ast.NewStringExpr(pos, v.Literal)
	case token.NIL:
		p.advance()
		return ast.NewNilExpr(pos)
	case token.TRUE:
		p.advance()
		return ast.NewTrueExpr(pos)
	case token.FALSE:
		p.advance()
		return ast.NewFalseExpr(pos)
	case token.DOTS:
		p.advance()
		return ast.NewVarArgExpr(pos)
	case token.Type('{'):
		return p.parseTableConstructor()
	case token.FUNCTION:
		p.advance()
		return p.parseFuncBody(false)
	default:
		return p.parseSuffixedExpr()
	}
}

func (p *parser) parseSuffixedExpr() ast.Expr {
	return p.continueSuffixedExpr(p.parsePrimaryExpr())
}

func (p *parser) parsePrimaryExpr() ast.Expr {
	if p.err != nil {
		return ast.NewNilExpr(p.pos())
	}
	switch p.tok.Type {
	case token.NAME:
		return p.parseName()
	case token.Type('('):
		pos := p.pos()
		p.advance()
		inner := p.parseExpr()
		p.expect(token.Type(')'))
		return ast.NewParenExpr(pos, inner)
	default:
		p.errorf("unexpected symbol near '%s'", p.tok)
		return ast.NewNilExpr(p.pos())
	}
}

func (p *parser) continueSuffixedExpr(expr ast.Expr) ast.Expr {
	for {
		if p.err != nil {
			return expr
		}
		switch {
		case p.check(token.Type('.')):
			pos := p.pos()
			p.advance()
			field := p.parseName()
			expr = ast.NewFieldExpr(pos, expr, field.Name)
		case p.check(token.Type('[')):
			pos := p.pos()
			p.advance()
			key := p.parseExpr()
			p.expect(token.Type(']'))
			expr = ast.NewIndexExpr(pos, expr, key)
		case p.check(token.Type(':')):
			pos := p.pos()
			p.advance()
			method := p.parseName()
			args := p.parseFuncArgs()
			expr = ast.NewMethodCallExpr(pos, expr, method.Name, args)
		case p.check(token.Type('(')) || p.check(token.STRING) || p.check(token.Type('{')):
			pos := p.pos()
			args := p.parseFuncArgs()
			expr = ast.NewFuncCallExpr(pos, expr, args)
		default:
			return expr
		}
	}
}

func (p *parser) parseFuncArgs() []ast.Expr {
	switch p.tok.Type {
	case token.Type('('):
		p.advance()
		var args []ast.Expr
		if !p.check(token.Type(')')) {
			args = p.parseExprList()
		}
		p.expect(token.Type(')'))
		return args
	case token.Type('{'):
		return []ast.Expr{p.parseTableConstructor()}
	case token.STRING:
		pos := p.pos()
		v := p.tok
		p.advance()
		return []ast.Expr{ast.NewStringExpr(pos, v.Literal)}
	default:
		p.errorf("function arguments expected near '%s'", p.tok)
		return nil
	}
}

func (p *parser) parseFuncBody(isMethod bool) *ast.FuncExpr {
	pos := p.pos()
	p.expect(token.Type('('))

	var params []*ast.NameExpr
	vararg := false
	varargName := ""

	if isMethod {
		params = append(params, ast.NewNameExpr(pos, "self"))
	}

	if !p.check(token.Type(')')) {
		for {
			if p.check(token.DOTS) {
				vararg = true
				p.advance()
				// Lua 5.5: ... can optionally be followed by a name
				if p.check(token.NAME) {
					varargName = p.tok.Literal
					p.advance()
				}
				break
			}
			params = append(params, p.parseName())
			if !p.match(token.Type(',')) {
				break
			}
		}
	}
	p.expect(token.Type(')'))
	body := p.parseBlock()
	p.expect(token.END)
	return ast.NewFuncExpr(pos, params, vararg, varargName, body)
}

func (p *parser) parseTableConstructor() *ast.TableConstructor {
	pos := p.pos()
	p.expect(token.Type('{'))
	var fields []*ast.TableField
	for !p.check(token.Type('}')) && !p.check(token.EOS) {
		if p.err != nil {
			break
		}
		fields = append(fields, p.parseField())
		if !p.match(token.Type(',')) && !p.match(token.Type(';')) {
			break
		}
	}
	p.expect(token.Type('}'))
	return ast.NewTableConstructor(pos, fields)
}

// parseField: '[' expr ']' '=' expr | NAME '=' expr | expr
func (p *parser) parseField() *ast.TableField {
	pos := p.pos()

	// [expr] = expr
	if p.check(token.Type('[')) {
		p.advance()
		key := p.parseExpr()
		p.expect(token.Type(']'))
		p.expect(token.Type('='))
		val := p.parseExpr()
		return ast.NewTableField(pos, key, val)
	}

	// NAME '=' expr  (lookahead: consume NAME, check for '=')
	if p.check(token.NAME) {
		nameTok := p.tok
		p.advance() // consume NAME
		if p.check(token.Type('=')) {
			// Record field
			p.advance() // skip '='
			val := p.parseExpr()
			key := ast.NewStringExpr(pos, nameTok.Literal)
			return ast.NewTableField(pos, key, val)
		}
		// Not a record field — build the expression starting from the name
		nameExpr := ast.NewNameExpr(pos, nameTok.Literal)
		expr := p.continueSuffixedExpr(nameExpr)
		expr = p.continueBinExpr(expr, 0)
		return ast.NewTableField(pos, nil, expr)
	}

	// List expression
	val := p.parseExpr()
	return ast.NewTableField(pos, nil, val)
}

// continueBinExpr handles residual binary operators after partial expression parsing.
func (p *parser) continueBinExpr(expr ast.Expr, limit int) ast.Expr {
	for {
		op := p.getBinop()
		if op == "" {
			break
		}
		pri := priorities[op]
		if pri.left <= limit {
			break
		}
		pos := p.pos()
		p.advance()
		right := p.parseSubExpr(pri.right)
		expr = ast.NewBinopExpr(pos, op, expr, right)
	}
	return expr
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func (p *parser) parseName() *ast.NameExpr {
	pos := p.pos()
	tok, _ := p.expect(token.NAME)
	return ast.NewNameExpr(pos, tok.Literal)
}

func (p *parser) parseExprList() []ast.Expr {
	exprs := []ast.Expr{p.parseExpr()}
	for p.match(token.Type(',')) {
		exprs = append(exprs, p.parseExpr())
	}
	return exprs
}
