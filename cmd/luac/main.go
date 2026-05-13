// Command luac compiles Lua source files and prints bytecode disassembly.
//
// Usage:
//
//	go run ./cmd/luac [flags] [file.lua ...]
//
// If no files are given, reads from stdin.
//
// Flags:
//
//	-e string    compile the given string instead of files
//	-tokens      also dump the token stream
//	-ast         also dump the AST before bytecode
//	-full        show full details (constants, locals, upvalues)
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/iceisfun/golua/v1/ast"
	astpkg "github.com/iceisfun/golua/v1/ast"
	"github.com/iceisfun/golua/v1/compiler"
	"github.com/iceisfun/golua/v1/lexer"
	"github.com/iceisfun/golua/v1/parser"
	"github.com/iceisfun/golua/v1/token"
)

var (
	exprFlag   = flag.String("e", "", "compile the given string")
	tokensFlag = flag.Bool("tokens", false, "dump token stream before bytecode")
	astFlag    = flag.Bool("ast", false, "dump AST before bytecode")
)

func main() {
	flag.Parse()

	if *exprFlag != "" {
		run("<string>", *exprFlag)
		return
	}

	args := flag.Args()
	if len(args) == 0 {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			fatal("reading stdin: %v", err)
		}
		run("<stdin>", string(data))
		return
	}

	for _, path := range args {
		data, err := os.ReadFile(path)
		if err != nil {
			fatal("reading %s: %v", path, err)
		}
		if len(args) > 1 {
			fmt.Printf("=== %s ===\n", path)
		}
		run(path, string(data))
	}
}

func run(source, input string) {
	if *tokensFlag {
		dumpTokens(source, input)
		fmt.Println()
	}

	block, err := parser.Parse(source, input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse error: %v\n", err)
		os.Exit(1)
	}

	if *astFlag {
		fmt.Println("--- AST ---")
		astpkg.Dump(os.Stdout, block)
		fmt.Println()
	}

	proto, err := compiler.Compile(source, block)
	if err != nil {
		fmt.Fprintf(os.Stderr, "compile error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("--- Bytecode ---")
	proto.Dump(os.Stdout)
}

func dumpTokens(source, input string) {
	fmt.Println("--- Tokens ---")
	lex := lexer.New(source, input, true)
	for {
		tok, err := lex.Next()
		if err != nil {
			fmt.Fprintf(os.Stderr, "lex error: %v\n", err)
			return
		}
		fmt.Printf("  %-20s %s\n", tok, posStr(tok.Pos))
		if tok.Type == token.EOS {
			break
		}
	}
}

func posStr(p token.Pos) string {
	parts := []string{}
	if p.Source != "" {
		parts = append(parts, p.Source)
	}
	parts = append(parts, fmt.Sprintf("%d:%d", p.Line, p.Column))
	return strings.Join(parts, ":")
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

// Suppress unused import for the ast dump interface
var _ = ast.Dump
