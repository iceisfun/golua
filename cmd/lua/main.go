package main

import (
	"fmt"
	"os"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
	"github.com/iceisfun/golua/stdlib"
	"github.com/iceisfun/golua/vm"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: lua <script.lua> [args...]")
		os.Exit(1)
	}

	filename := os.Args[1]

	// Read the source file
	src, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Parse
	block, err := parser.Parse(filename, string(src))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Parse error: %v\n", err)
		os.Exit(1)
	}

	// Compile
	proto, err := compiler.Compile(filename, block)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Compile error: %v\n", err)
		os.Exit(1)
	}

	// Create VM and register standard library
	v := vm.New()
	stdlib.Open(v)

	// Set command line arguments
	args := vm.NewEmptyTable()
	args.SetInt(0, vm.NewString(filename))
	for i, arg := range os.Args[2:] {
		args.SetInt(i+1, vm.NewString(arg))
	}
	v.SetGlobal("arg", vm.NewTable(args))

	// Run
	_, err = v.Run(proto)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Runtime error: %v\n", err)
		os.Exit(1)
	}
}
