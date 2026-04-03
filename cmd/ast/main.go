// Command ast parses Lua source files and prints their AST for debugging.
//
// Usage:
//
//	go run ./cmd/ast [flags] [file.lua ...]
//
// If no files are given, reads from stdin.
// Use -e to parse a string directly.
//
// Flags:
//
//	-e string    parse the given string instead of files
//	-pos         include source positions on each node
//	-tokens      also dump the token stream before the AST
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/iceisfun/golua/v2/ast"
	"github.com/iceisfun/golua/v2/lexer"
	"github.com/iceisfun/golua/v2/parser"
	"github.com/iceisfun/golua/v2/token"
)

var (
	exprFlag   = flag.String("e", "", "parse the given string")
	posFlag    = flag.Bool("pos", false, "include source positions on each node")
	tokensFlag = flag.Bool("tokens", false, "dump token stream before AST")
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

	ast.DumpWith(os.Stdout, block, ast.DumpOptions{ShowPos: *posFlag})
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
	fmt.Println("--- AST ---")
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
