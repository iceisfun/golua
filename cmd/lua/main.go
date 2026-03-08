package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
	"github.com/iceisfun/golua/stdlib"
	"github.com/iceisfun/golua/vm"
)

func main() {
	var timeoutMs int
	var evalCode string
	var testMode bool
	args := os.Args[1:]

	// Parse flags
	for len(args) > 0 {
		switch args[0] {
		case "--timeout":
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
		case "-e", "--e":
			if len(args) < 2 {
				fmt.Fprintln(os.Stderr, "-e requires a string argument")
				os.Exit(1)
			}
			evalCode = args[1]
			args = args[2:]
		case "--test":
			testMode = true
			args = args[1:]
		default:
			goto done
		}
	}
done:

	var source string
	var name string
	var scriptArgs []string

	if evalCode != "" {
		source = evalCode
		name = "=(command line)"
		scriptArgs = args
	} else {
		if len(args) < 1 {
			fmt.Fprintln(os.Stderr, "Usage: lua [--timeout <ms>] [-e <code>] [--test] [<script.lua> [args...]]")
			os.Exit(1)
		}
		filename := args[0]
		scriptArgs = args[1:]
		src, err := os.ReadFile(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
			os.Exit(1)
		}
		source = string(src)
		// Strip UTF-8 BOM if present (like Lua 5.4)
		if len(source) >= 3 && source[0] == 0xEF && source[1] == 0xBB && source[2] == 0xBF {
			source = source[3:]
		}
		name = "@" + filename
	}

	// Parse — use the display name (without '@') for parser error messages
	displayName := name
	if len(displayName) > 0 && displayName[0] == '@' {
		displayName = displayName[1:]
	} else if len(displayName) > 0 && displayName[0] == '=' {
		displayName = displayName[1:]
	}
	// Determine program name for error messages (like Lua 5.4 uses argv[0])
	progName := filepath.Base(os.Args[0])

	block, err := parser.Parse(displayName, source)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", progName, err)
		os.Exit(1)
	}

	// Compile — use the raw name so proto.Source stores it with the '@' prefix
	proto, err := compiler.Compile(name, block)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", progName, err)
		os.Exit(1)
	}

	// Create VM and register standard library
	v := vm.New()
	v.SetOsProvider(vm.NewDefaultOsProvider())

	// Determine script directory for IO provider
	scriptDir := "."
	if name != "=(command line)" {
		// Strip '@' prefix for filesystem operations
		fsName := name
		if len(fsName) > 0 && fsName[0] == '@' {
			fsName = fsName[1:]
		}
		if d := filepath.Dir(fsName); d != "" {
			scriptDir = d
		}
	}

	// Always set core providers for CLI usage
	v.SetCodeProvider(vm.NewDirCodeProvider(".", vm.LuaLoaderCaps{
		AllowLoadfile: true,
		AllowDofile:   true,
	}))
	v.SetIoProvider(vm.NewFullIoProvider(scriptDir))
	v.SetExecProvider(vm.NewDefaultExecProvider())
	v.SetExitHandler(vm.NewDefaultExitHandler())

	if testMode {
		v.SetDebugProvider(vm.NewDefaultDebugProvider())
	}
	stdlib.Open(v)

	// Set timeout context if requested
	if timeoutMs > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
		defer cancel()
		v.SetContext(ctx)
	}

	// Set command line arguments — use displayName (without '@' prefix)
	luaArgs := vm.NewEmptyTable()
	luaArgs.SetInt(0, vm.NewString(displayName))
	for i, arg := range scriptArgs {
		luaArgs.SetInt(i+1, vm.NewString(arg))
	}
	v.SetGlobal("arg", vm.NewTable(luaArgs))

	// Run — recover LuaExitError for os.exit support
	var exitCode int
	func() {
		defer func() {
			if r := recover(); r != nil {
				if exitErr, ok := r.(*vm.LuaExitError); ok {
					exitCode = exitErr.Code
					return
				}
				// Re-panic for unexpected panics
				panic(r)
			}
		}()
		_, err = v.Run(proto)
	}()

	if err != nil {
		// Format the error like Lua 5.4
		if le, ok := err.(*vm.LuaError); ok {
			msg := formatLuaError(le.Value)
			fmt.Fprintf(os.Stderr, "%s\n", msg)
		} else {
			fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		}
		// Print stack traceback
		fmt.Fprintln(os.Stderr, "stack traceback:")
		fmt.Fprintln(os.Stderr, "\t[C]: in ?")
		os.Exit(1)
	}

	if exitCode != 0 {
		os.Exit(exitCode)
	}
}

// formatLuaError formats a Lua error value for display, matching Lua 5.4.
// Strings are printed as-is, numbers with .0 suffix for floats,
// other types are wrapped in "(error object is a TYPE value)".
func formatLuaError(val vm.Value) string {
	if val.IsString() {
		return val.AsString()
	}
	if val.IsInt() {
		return fmt.Sprintf("%d", val.AsInt())
	}
	if val.IsFloat() {
		f := val.AsFloat()
		s := fmt.Sprintf("%g", f)
		// Ensure float has .0 suffix
		if !math.IsInf(f, 0) && !math.IsNaN(f) && !strings.Contains(s, ".") && !strings.Contains(s, "e") {
			s += ".0"
		}
		return s
	}
	return fmt.Sprintf("(error object is a %s value)", val.Type())
}
