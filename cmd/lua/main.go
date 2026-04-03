package main

import (
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/iceisfun/golua/v2/compiler"
	"github.com/iceisfun/golua/v2/parser"
	"github.com/iceisfun/golua/v2/stdlib"
	"github.com/iceisfun/golua/v2/vm"
)

func main() {
	os.Exit(runCLI(os.Args, os.Stderr))
}

func configureCLIProviders(v *vm.VM, testMode bool, scriptDir string) {
	v.SetCodeProvider(vm.NewDirCodeProvider(".", vm.LuaLoaderCaps{
		AllowLoadfile: true,
		AllowDofile:   true,
	}))
	if testMode {
		v.SetIoProvider(vm.NewJailedIoProvider(scriptDir))
		v.SetDebugProvider(vm.NewDefaultDebugProvider())
		return
	}
	v.SetIoProvider(vm.NewFullIoProvider(scriptDir))
	v.SetExecProvider(vm.NewDefaultExecProvider())
	v.SetExitHandler(vm.NewDefaultExitHandler())
	v.SetDebugProvider(vm.NewDefaultDebugProvider())
}

func runCLI(argv []string, stderr io.Writer) int {
	var timeoutMs int
	var evalCode string
	var testMode bool
	var gcStepInterval int
	args := argv[1:]

	// Parse flags
	for len(args) > 0 {
		switch args[0] {
		case "--":
			args = args[1:]
			goto done
		case "--timeout":
			if len(args) < 2 {
				fmt.Fprintln(stderr, "--timeout requires a value in milliseconds")
				return 1
			}
			var err error
			timeoutMs, err = strconv.Atoi(args[1])
			if err != nil || timeoutMs <= 0 {
				fmt.Fprintln(stderr, "--timeout value must be a positive integer (milliseconds)")
				return 1
			}
			args = args[2:]
		case "-e", "--e":
			if len(args) < 2 {
				fmt.Fprintln(stderr, "-e requires a string argument")
				return 1
			}
			evalCode = args[1]
			args = args[2:]
		case "--test":
			testMode = true
			args = args[1:]
		case "--gc-step":
			if len(args) < 2 {
				fmt.Fprintln(stderr, "--gc-step requires a value (instruction interval)")
				return 1
			}
			var err error
			gcStepInterval, err = strconv.Atoi(args[1])
			if err != nil || gcStepInterval <= 0 {
				fmt.Fprintln(stderr, "--gc-step value must be a positive integer")
				return 1
			}
			args = args[2:]
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
			fmt.Fprintln(stderr, "Usage: lua [--timeout <ms>] [-e <code>] [--test] [<script.lua> [args...]]")
			return 1
		}
		filename := args[0]
		scriptArgs = args[1:]
		src, err := os.ReadFile(filename)
		if err != nil {
			fmt.Fprintf(stderr, "Error reading file: %v\n", err)
			return 1
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
	progName := filepath.Base(argv[0])

	block, err := parser.Parse(displayName, source)
	if err != nil {
		fmt.Fprintf(stderr, "%s: %v\n", progName, err)
		return 1
	}

	// Compile — use the raw name so proto.Source stores it with the '@' prefix.
	proto, err := compiler.Compile(name, block)
	if err != nil {
		fmt.Fprintf(stderr, "%s: %v\n", progName, err)
		return 1
	}

	// Create VM and register standard library
	ctx := context.Background()
	var cancel context.CancelFunc
	if timeoutMs > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
		defer cancel()
	}

	var vmOpts []vm.VMOption
	vmOpts = append(vmOpts, vm.WithContext(ctx))
	if gcStepInterval > 0 {
		vmOpts = append(vmOpts, vm.WithLimits(vm.Limits{GCStepInterval: gcStepInterval}))
	}
	v := vm.New(vmOpts...)
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

	configureCLIProviders(v, testMode, scriptDir)
	stdlib.Open(v)

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

	v.Close(context.Background())

	if err != nil {
		// Format error like Lua 5.4's msghandler + report in lua.c.
		// msghandler logic:
		//   1. If error is a string/number, append traceback
		//   2. If error has __tostring returning string, use it (no traceback)
		//   3. If __tostring errors, that error propagates (shown recursively)
		//   4. Otherwise "(error object is a TYPE value)" with traceback
		reportLuaError(v, err, progName, stderr)
		return 1
	}

	if exitCode != 0 {
		return exitCode
	}
	return 0
}

// reportLuaError formats and prints a Lua error matching lua.c's msghandler+report.
// The logic mirrors C Lua 5.4:
//   - String/number errors: print message + traceback
//   - __tostring returning string: print that string, NO traceback
//   - __tostring errors: recurse with the inner error
//   - Otherwise: "(error object is a TYPE value)" + traceback
func reportLuaError(v *vm.VM, err error, progName string, w io.Writer) {
	le, ok := err.(*vm.LuaError)
	if !ok {
		fmt.Fprintf(w, "%s: %s\n", progName, err.Error())
		fmt.Fprintln(w, v.TracebackFromLastError("", 0))
		return
	}
	val := le.Value

	// Step 1: string/number — like lua_tostring succeeding
	if val.IsString() || val.IsNumber() {
		fmt.Fprintf(w, "%s: %s\n", progName, formatLuaValue(val))
		fmt.Fprintln(w, v.TracebackFromLastError("", 0))
		return
	}

	// Step 2: try __tostring metamethod (like luaL_callmeta)
	var mt vm.LuaTable
	if val.IsTable() {
		mt = val.AsTable().Metatable()
	} else if ud := val.AsUserdata(); ud != nil {
		mt = ud.Metatable()
	}
	if mt == nil {
		mt = v.GetTypeMeta(val)
	}
	if mt != nil {
		if ts := mt.Get(vm.NewString("__tostring")); !ts.IsNil() {
			// Save the outer traceback before ProtectedCall overwrites it.
			// In C Lua, __tostring runs unprotected inside the message
			// handler, so if it errors the combined traceback includes
			// both the inner __tostring frames and the outer error() site.
			outerTb := v.TracebackFromLastError("", 0)
			// Explicitly clear the snapshot so the inner ProtectedCall
			// can capture fresh inner frames (TracebackFromLastError
			// doesn't consume when callStack is empty — our case after Run).
			v.ClearLastErrorCallStack()
			results, callErr := v.ProtectedCall(ts, []vm.Value{val})
			if callErr != nil {
				// __tostring errored — report the inner error first,
				// then append the outer traceback frames to match
				// C Lua's combined traceback.
				reportLuaError(v, callErr, progName, w)
				if outerTb != "" {
					// Append outer frames (skip "stack traceback:" header).
					if idx := strings.Index(outerTb, "\n"); idx >= 0 {
						fmt.Fprint(w, outerTb[idx+1:])
						fmt.Fprintln(w)
					}
				}
				return
			}
			if len(results) > 0 && results[0].IsString() {
				// __tostring returned a string — print WITHOUT traceback
				fmt.Fprintf(w, "%s: %s\n", progName, results[0].AsString())
				return
			}
			// __tostring returned non-string — fall through with outer traceback
			fmt.Fprintf(w, "%s: (error object is a %s value)\n", progName, val.Type())
			fmt.Fprintln(w, outerTb)
			return
		}
	}

	// Step 3: fallback — "(error object is a TYPE value)" + traceback
	fmt.Fprintf(w, "%s: (error object is a %s value)\n", progName, val.Type())
	fmt.Fprintln(w, v.TracebackFromLastError("", 0))
}

// formatLuaValue formats a Lua string/number value for display.
func formatLuaValue(val vm.Value) string {
	if val.IsString() {
		return val.AsString()
	}
	if val.IsInt() {
		return fmt.Sprintf("%d", val.AsInt())
	}
	if val.IsFloat() {
		f := val.AsFloat()
		s := fmt.Sprintf("%g", f)
		if !math.IsInf(f, 0) && !math.IsNaN(f) && !strings.Contains(s, ".") && !strings.Contains(s, "e") {
			s += ".0"
		}
		return s
	}
	return val.String()
}

