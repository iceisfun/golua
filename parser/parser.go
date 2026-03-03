// Package parser implements a recursive descent parser for Lua 5.4, producing
// an abstract syntax tree (AST) from source text.
//
// The parser uses the lexer to tokenize input and builds an AST following
// Lua 5.4's grammar with precedence-climbing for expressions. It supports
// all Lua 5.4 syntax including goto/labels, local attributes (<const>, <close>),
// and shebang line stripping for script loading.
//
// Lua 5.4 Reference: §3 – The Language (§3.3 Statements, §3.4 Expressions).
package parser

import (
	"fmt"

	"github.com/iceisfun/golua/ast"
	"github.com/iceisfun/golua/lexer"
	"github.com/iceisfun/golua/token"
)

// ParsePartial parses the given Lua source and returns a partial AST block
// alongside any error. Unlike Parse, it never returns a nil block — on error,
// the block contains all statements that were successfully parsed before the
// first error.
func ParsePartial(source, input string) (*ast.Block, error) {
	p := &parser{
		lex:    lexer.New(source, input, true),
		source: source,
	}
	if err := p.advance(); err != nil {
		return &ast.Block{Start: token.Pos{Source: source, Line: 1, Column: 1}}, err
	}
	block := p.parseBlock()
	if p.err != nil {
		return block, p.err
	}
	if p.tok.Type != token.EOS {
		return block, p.errorf("unexpected symbol near '%s'", p.tok.Literal)
	}
	return block, nil
}

// Parse parses the given Lua source and returns the top-level block.
// An optional third argument controls shebang stripping (default true).
func Parse(source, input string, stripShebang ...bool) (*ast.Block, error) {
	strip := true
	if len(stripShebang) > 0 {
		strip = stripShebang[0]
	}
	p := &parser{
		lex:    lexer.New(source, input, strip),
		source: source,
	}
	if err := p.advance(); err != nil {
		return nil, err
	}
	block := p.parseBlock()
	if p.err != nil {
		return nil, p.err
	}
	if p.tok.Type != token.EOS {
		return nil, p.errorf("unexpected symbol near '%s'", p.tok.Literal)
	}
	return block, nil
}

// ---------------------------------------------------------------------------
// Parser state
// ---------------------------------------------------------------------------

type parser struct {
	lex    *lexer.Lexer
	tok    token.Token // current (lookahead) token
	source string
	err    error
}

func (p *parser) advance() error {
	tok, err := p.lex.Next()
	if err != nil {
		p.err = err
		return err
	}
	p.tok = tok
	return nil
}

func (p *parser) expect(typ token.Type) (token.Token, error) {
	if p.tok.Type != typ {
		return token.Token{}, p.errorf("'%s' expected near '%s'", typ, p.tok)
	}
	tok := p.tok
	if err := p.advance(); err != nil {
		return token.Token{}, err
	}
	return tok, nil
}

func (p *parser) check(typ token.Type) bool {
	return p.tok.Type == typ
}

func (p *parser) match(typ token.Type) bool {
	if p.tok.Type == typ {
		p.advance()
		return true
	}
	return false
}

func (p *parser) errorf(format string, args ...any) error {
	if p.err != nil {
		return p.err
	}
	p.err = &token.PosError{Pos: p.tok.Pos, Msg: fmt.Sprintf(format, args...)}
	return p.err
}

func (p *parser) pos() token.Pos { return p.tok.Pos }

// ---------------------------------------------------------------------------
// Block / statement list
// ---------------------------------------------------------------------------

func (p *parser) blockFollow(withUntil bool) bool {
	switch p.tok.Type {
	case token.ELSE, token.ELSEIF, token.END, token.EOS:
		return true
	case token.UNTIL:
		return withUntil
	default:
		return false
	}
}

func (p *parser) parseBlock() *ast.Block {
	block := &ast.Block{Start: p.pos()}
	for !p.blockFollow(true) {
		s := p.parseStatement()
		if s == nil {
			break
		}
		block.Stmts = append(block.Stmts, s)
		if p.err != nil {
			break
		}
	}
	return block
}

// ---------------------------------------------------------------------------
// Statements
// ---------------------------------------------------------------------------

func (p *parser) parseStatement() ast.Stmt {
	if p.err != nil {
		return nil
	}
	switch p.tok.Type {
	case token.Type(';'):
		s := ast.NewEmptyStmt(p.pos())
		p.advance()
		return s
	case token.IF:
		return p.parseIfStmt()
	case token.WHILE:
		return p.parseWhileStmt()
	case token.DO:
		return p.parseDoStmt()
	case token.FOR:
		return p.parseForStmt()
	case token.REPEAT:
		return p.parseRepeatStmt()
	case token.FUNCTION:
		return p.parseFuncStmt()
	case token.LOCAL:
		return p.parseLocalStmt()
	case token.GLOBAL:
		return p.parseGlobalStmt()
	case token.RETURN:
		return p.parseReturnStmt()
	case token.BREAK:
		s := ast.NewBreakStmt(p.pos())
		p.advance()
		return s
	case token.GOTO:
		return p.parseGotoStmt()
	case token.DBCOLON:
		return p.parseLabelStmt()
	default:
		return p.parseExprStat()
	}
}

func (p *parser) parseIfStmt() ast.Stmt {
	pos := p.pos()
	p.advance() // skip 'if'
	cond := p.parseExpr()
	p.expect(token.THEN)
	then := p.parseBlock()

	var elseifs []*ast.ElseIf
	for p.check(token.ELSEIF) {
		eiPos := p.pos()
		p.advance()
		eiCond := p.parseExpr()
		p.expect(token.THEN)
		eiThen := p.parseBlock()
		elseifs = append(elseifs, ast.NewElseIf(eiPos, eiCond, eiThen))
	}

	var elseb *ast.Block
	if p.match(token.ELSE) {
		elseb = p.parseBlock()
	}
	p.expect(token.END)
	return ast.NewIfStmt(pos, cond, then, elseifs, elseb)
}

func (p *parser) parseWhileStmt() ast.Stmt {
	pos := p.pos()
	p.advance()
	cond := p.parseExpr()
	p.expect(token.DO)
	body := p.parseBlock()
	p.expect(token.END)
	return ast.NewWhileStmt(pos, cond, body)
}

func (p *parser) parseDoStmt() ast.Stmt {
	pos := p.pos()
	p.advance()
	body := p.parseBlock()
	p.expect(token.END)
	return ast.NewDoStmt(pos, body)
}

func (p *parser) parseForStmt() ast.Stmt {
	pos := p.pos()
	p.advance() // skip 'for'
	name := p.parseName()
	if p.check(token.Type('=')) {
		return p.parseForNumStmt(pos, name)
	}
	return p.parseForInStmt(pos, name)
}

func (p *parser) parseForNumStmt(pos token.Pos, name *ast.NameExpr) ast.Stmt {
	p.expect(token.Type('='))
	start := p.parseExpr()
	p.expect(token.Type(','))
	stop := p.parseExpr()
	var step ast.Expr
	if p.match(token.Type(',')) {
		step = p.parseExpr()
	}
	p.expect(token.DO)
	body := p.parseBlock()
	p.expect(token.END)
	return ast.NewForNumStmt(pos, name, start, stop, step, body)
}

func (p *parser) parseForInStmt(pos token.Pos, firstName *ast.NameExpr) ast.Stmt {
	names := []*ast.NameExpr{firstName}
	for p.match(token.Type(',')) {
		names = append(names, p.parseName())
	}
	p.expect(token.IN)
	iters := p.parseExprList()
	p.expect(token.DO)
	body := p.parseBlock()
	p.expect(token.END)
	return ast.NewForInStmt(pos, names, iters, body)
}

func (p *parser) parseRepeatStmt() ast.Stmt {
	pos := p.pos()
	p.advance()
	body := p.parseBlock()
	p.expect(token.UNTIL)
	cond := p.parseExpr()
	return ast.NewRepeatStmt(pos, body, cond)
}

// parseFuncStmt: function funcname body
// funcname: NAME {'.' NAME} [':' NAME]
func (p *parser) parseFuncStmt() ast.Stmt {
	pos := p.pos()
	p.advance() // skip 'function'
	name := p.parseName()
	var nameExpr ast.Expr = name
	isMethod := false
	for p.match(token.Type('.')) {
		field := p.parseName()
		nameExpr = ast.NewFieldExpr(nameExpr.Pos(), nameExpr, field.Name)
	}
	if p.match(token.Type(':')) {
		method := p.parseName()
		nameExpr = ast.NewFieldExpr(nameExpr.Pos(), nameExpr, method.Name)
		isMethod = true
	}
	fn := p.parseFuncBody(isMethod)
	return ast.NewFuncStmt(pos, nameExpr, isMethod, fn)
}

func (p *parser) parseLocalStmt() ast.Stmt {
	pos := p.pos()
	p.advance() // skip 'local'

	if p.match(token.FUNCTION) {
		name := p.parseName()
		fn := p.parseFuncBody(false)
		return ast.NewLocalFuncStmt(pos, name, fn)
	}

	defAttrib := p.parseAttrib()
	names := []*ast.NameExpr{p.parseName()}
	attribs := []string{p.parseAttribOr(defAttrib)}
	for p.match(token.Type(',')) {
		names = append(names, p.parseName())
		attribs = append(attribs, p.parseAttribOr(defAttrib))
	}
	var values []ast.Expr
	if p.match(token.Type('=')) {
		values = p.parseExprList()
	}
	return ast.NewLocalStmt(pos, names, attribs, values)
}

func (p *parser) parseGlobalStmt() ast.Stmt {
	pos := p.pos()
	p.advance() // skip 'global'

	if p.match(token.FUNCTION) {
		name := p.parseName()
		fn := p.parseFuncBody(false)
		return ast.NewGlobalFuncStmt(pos, name, fn)
	}

	defAttrib := p.parseAttrib()
	if p.match(token.Type('*')) {
		return ast.NewGlobalStarStmt(pos, defAttrib)
	}

	names := []*ast.NameExpr{p.parseName()}
	attribs := []string{p.parseAttribOr(defAttrib)}
	for p.match(token.Type(',')) {
		names = append(names, p.parseName())
		attribs = append(attribs, p.parseAttribOr(defAttrib))
	}
	var values []ast.Expr
	if p.match(token.Type('=')) {
		values = p.parseExprList()
	}
	return ast.NewGlobalStmt(pos, names, attribs, values)
}

func (p *parser) parseAttrib() string {
	if p.match(token.Type('<')) {
		tok, _ := p.expect(token.NAME)
		if tok.Literal != "const" && tok.Literal != "close" {
			p.err = &token.PosError{Pos: tok.Pos, Msg: fmt.Sprintf("unknown attribute '%s'", tok.Literal)}
		}
		p.expect(token.Type('>'))
		return tok.Literal
	}
	return ""
}

func (p *parser) parseAttribOr(def string) string {
	if a := p.parseAttrib(); a != "" {
		return a
	}
	return def
}

func (p *parser) parseReturnStmt() ast.Stmt {
	pos := p.pos()
	p.advance() // skip 'return'
	var values []ast.Expr
	if !p.blockFollow(true) && !p.check(token.Type(';')) {
		values = p.parseExprList()
	}
	p.match(token.Type(';'))
	return ast.NewReturnStmt(pos, values)
}

func (p *parser) parseGotoStmt() ast.Stmt {
	pos := p.pos()
	p.advance() // skip 'goto'
	name, _ := p.expect(token.NAME)
	return ast.NewGotoStmt(pos, name.Literal)
}

func (p *parser) parseLabelStmt() ast.Stmt {
	pos := p.pos()
	p.advance() // skip '::'
	name, _ := p.expect(token.NAME)
	p.expect(token.DBCOLON)
	return ast.NewLabelStmt(pos, name.Literal)
}

func (p *parser) parseExprStat() ast.Stmt {
	pos := p.pos()
	expr := p.parseSuffixedExpr()

	if p.check(token.Type('=')) || p.check(token.Type(',')) {
		targets := []ast.Expr{expr}
		for p.match(token.Type(',')) {
			targets = append(targets, p.parseSuffixedExpr())
		}
		p.expect(token.Type('='))
		values := p.parseExprList()
		return ast.NewAssignStmt(pos, targets, values)
	}

	switch expr.(type) {
	case *ast.FuncCallExpr, *ast.MethodCallExpr:
		return ast.NewExprStmt(pos, expr)
	default:
		p.errorf("syntax error: unexpected expression statement")
		return nil
	}
}

