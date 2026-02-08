package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
	"github.com/iceisfun/golua/stdlib"
	"github.com/iceisfun/golua/vm"
)

func main() {
	var timeoutMs int
	args := os.Args[1:]

	// Parse --timeout flag
	for len(args) > 0 {
		if args[0] == "--timeout" {
			if len(args) < 2 {
				fmt.Fprintln(os.Stderr, "--timeout requires a value in milliseconds")
				os.Exit(1)
			}
			var err error
			timeoutMs, err = strconv.Atoi(args[1])
			if err != nil || timeoutMs <= 0 {
				fmt.Fprintln(os.Stderr, "--timeout value must be a positive integer (milliseconds)")
				os.Exit(1)
			}
			args = args[2:]
		} else {
			break
		}
	}

	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: lua [--timeout <ms>] <script.lua> [args...]")
		os.Exit(1)
	}

	filename := args[0]
	scriptArgs := args[1:]

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
	v.SetOsProvider(vm.NewDefaultOsProvider())
	stdlib.Open(v)

	// Set timeout context if requested
	if timeoutMs > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
		defer cancel()
		v.SetContext(ctx)
	}

	// Set command line arguments
	luaArgs := vm.NewEmptyTable()
	luaArgs.SetInt(0, vm.NewString(filename))
	for i, arg := range scriptArgs {
		luaArgs.SetInt(i+1, vm.NewString(arg))
	}
	v.SetGlobal("arg", vm.NewTable(luaArgs))

	// Run
	_, err = v.Run(proto)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Runtime error: %v\n", err)
		os.Exit(1)
	}
}
