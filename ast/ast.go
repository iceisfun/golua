// Package ast defines the abstract syntax tree (AST) node types produced by
// the parser and consumed by the compiler.
//
// Every node carries a source position (token.Pos) for error reporting.
// The tree is organized into two interface hierarchies: Expr for expressions
// and Stmt for statements, both extending the common Node interface.
//
// Each concrete node type has a corresponding constructor (e.g. NewIfStmt)
// that initializes all fields. The AST is semantically neutral — it represents
// syntactic structure only and carries no type or scope information.
//
// Lua 5.4 Reference: §3 – The Language (§3.3 Statements, §3.4 Expressions).
package ast

import "github.com/iceisfun/golua/v2/token"

// ---------------------------------------------------------------------------
// Interfaces
// ---------------------------------------------------------------------------

// Node is the common interface for every AST node. Pos returns the position of
// the node's first token; End returns the position just past its last token.
type Node interface {
	Pos() token.Pos
	End() token.Pos
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
	Start   token.Pos
	EndLine int // line of the closing keyword or EOF (0 if unknown)
	Stmts   []Stmt
}

func (b *Block) Pos() token.Pos { return b.Start }
