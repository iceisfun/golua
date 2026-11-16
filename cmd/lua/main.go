package main

import (
	"context"
	"errors"
	"fmt"
	"io"
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
	// Enable io.popen alongside os.execute: both are subprocess-spawning
	// capabilities and reference Lua exposes both at the standalone-interpreter
	// trust level. io.popen is gated by the process provider (separate from the
	// exec provider that backs os.execute), so it must be set explicitly.
	v.SetProcessProvider(vm.NewDefaultProcessProvider())
	v.SetExitHandler(vm.NewDefaultExitHandler())
	v.SetDebugProvider(vm.NewDefaultDebugProvider())
}

// Streams used for the standalone interpreter's own output (-v) and for
// reading a script from standard input. Variables so tests can substitute them.
var (
	cliStdout io.Writer = os.Stdout
	cliStdin  io.Reader = os.Stdin
)

// luaVersionBanner is what -v prints, like reference lua.c's printversion.
const luaVersionBanner = "Lua 5.5.1  Copyright (C) 1994-2026 Lua.org, PUC-Rio"

const cliUsage = `usage: %s [options] [script [args]]
Available options are:
  -e stat   execute string 'stat'
  -i        enter interactive mode after executing 'script'
            (limited: standard input is then run as a script)
  -l mod    require library 'mod' into global 'mod'
  -l g=mod  require library 'mod' into global 'g'
  -v        show version information
  -E        ignore environment variables
  -W        turn warnings on
  --        stop handling options
  -         stop handling options and execute stdin
  --timeout <ms>   abort the run after the given number of milliseconds
  --gc-step <n>    set the collector step interval (instructions)
  --test           run with the jailed test-mode providers
`

// cliAction is one -e or -l option, run in command-line order.
type cliAction struct {
	kind byte // 'e' or 'l'
	arg  string
}

// cliOptions is the result of collecting the command line, like the flags
// gathered by reference lua.c's collectargs.
type cliOptions struct {
	hasI, hasV, hasE, hasW bool
	actions                []cliAction
	hasEStat               bool // at least one -e was given
	script                 int  // index in argv of the script operand, len(argv) if none
	timeoutMs              int
	gcStep                 int
	testMode               bool
}

// collectArgs scans the options before the script, like lua.c's collectargs.
// A non-empty return is the error message to report before the usage text.
func collectArgs(argv []string) (cliOptions, string) {
	o := cliOptions{script: len(argv)}
	// needArg returns the argument of an option that takes one, attached
	// ("-ecode") or in the next word.
	needArg := func(i int, opt string) (string, int, string) {
		if len(argv[i]) > len(opt) {
			return argv[i][len(opt):], i, ""
		}
		if i+1 < len(argv) {
			return argv[i+1], i + 1, ""
		}
		return "", i, fmt.Sprintf("'%s' needs argument", opt)
	}
	i := 1
	for ; i < len(argv); i++ {
		a := argv[i]
		if a == "" || a[0] != '-' {
			o.script = i // not an option: the script name
			return o, ""
		}
		switch {
		case a == "--":
			o.script = i + 1
			return o, ""
		case a == "-":
			o.script = i // stdin script
			return o, ""
		case a == "--timeout" || a == "--gc-step":
			if i+1 >= len(argv) {
				return o, fmt.Sprintf("'%s' needs argument", a)
			}
			n, err := strconv.Atoi(argv[i+1])
			if err != nil || n <= 0 {
				return o, fmt.Sprintf("'%s' value must be a positive integer", a)
			}
			if a == "--timeout" {
				o.timeoutMs = n
			} else {
				o.gcStep = n
			}
			i++
		case a == "--test":
			o.testMode = true
		case a == "--e": // historical spelling
			val, ni, msg := needArg(i, "--e")
			if msg != "" {
				return o, msg
			}
			o.actions = append(o.actions, cliAction{'e', val})
			o.hasEStat = true
			i = ni
		case a == "-E":
			o.hasE = true
		case a == "-W":
			o.hasW = true
		case a == "-v":
			o.hasV = true
		case a == "-i":
			o.hasI = true
			o.hasV = true
		case a[1] == 'e' || a[1] == 'l':
			opt := a[:2]
			val, ni, msg := needArg(i, opt)
			if msg != "" {
				return o, msg
			}
			if opt == "-e" {
				o.hasEStat = true
			} else if val == "" || val[0] == '-' {
				return o, "'-l' needs argument"
			}
			o.actions = append(o.actions, cliAction{opt[1], val})
			i = ni
		default:
			return o, fmt.Sprintf("unrecognized option '%s'", a)
		}
	}
	return o, ""
}

func runCLI(argv []string, stderr io.Writer) int {
	progName := filepath.Base(argv[0])
	opts, bad := collectArgs(argv)
	if bad != "" {
		fmt.Fprintf(stderr, "%s: %s\n", progName, bad)
		fmt.Fprintf(stderr, cliUsage, progName)
		return 1
	}
	if opts.hasV {
		fmt.Fprintln(cliStdout, luaVersionBanner)
	}

	scriptName := ""
	if opts.script < len(argv) {
		scriptName = argv[opts.script]
	}
	fromStdin := scriptName == "-"

	// Create VM and register standard library
	ctx := context.Background()
	if opts.timeoutMs > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(opts.timeoutMs)*time.Millisecond)
		defer cancel()
	}

	var vmOpts []vm.VMOption
	vmOpts = append(vmOpts, vm.WithContext(ctx))
	if opts.gcStep > 0 {
		vmOpts = append(vmOpts, vm.WithLimits(vm.Limits{GCStepInterval: opts.gcStep}))
	}
	v := vm.New(vmOpts...)
	defer v.Close(context.Background())

	// -E (LUA_NOENV): LUA_PATH and LUA_CPATH are read while the package
	// library opens, so hide the path variables from it only.
	if opts.hasE {
		v.SetOsProvider(vm.NewFilteredOsProvider(func(name string) bool {
			return !strings.HasPrefix(name, "LUA_PATH") && !strings.HasPrefix(name, "LUA_CPATH")
		}))
	} else {
		v.SetOsProvider(vm.NewDefaultOsProvider())
	}

	// Determine script directory for IO provider
	scriptDir := "."
	if scriptName != "" && !fromStdin {
		if d := filepath.Dir(scriptName); d != "" {
			scriptDir = d
		}
	}

	configureCLIProviders(v, opts.testMode, scriptDir)
	stdlib.Open(v)
	if opts.hasW {
		v.SetWarnEnabled(true)
	}

	// Set command line arguments. Like reference Lua (manual §7, lua.c
	// createargtable), argv[i] lands at arg[i - script]: the script name is
	// arg[0] and the interpreter and options precede it at negative
	// indices. With no script, the interpreter name is arg[0].
	scriptIdx := opts.script
	if scriptIdx >= len(argv) {
		scriptIdx = 0
	}
	luaArgs := vm.NewEmptyTable()
	for i, a := range argv {
		luaArgs.SetInt(i-scriptIdx, vm.NewString(a))
	}
	v.SetGlobal("arg", vm.NewTable(luaArgs))

	// finish reports err, if any, and gives the exit status for a run that
	// stops here. It reports whether the CLI must stop.
	finish := func(code int, exited bool, err error) (int, bool) {
		if err != nil {
			reportLuaError(v, err, progName, stderr)
			return 1, true
		}
		return code, exited
	}
	loadErr := func(err error) (int, bool) {
		fmt.Fprintf(stderr, "%s: %v\n", progName, err)
		return 1, true
	}

	// LUA_INIT_5_5 / LUA_INIT (reference handle_luainit): a value of the
	// form '@filename' runs the file, anything else runs as code — before
	// the main chunk. Errors abort the run like any script error. -E skips it.
	initName := "=LUA_INIT_5_5"
	initVal, haveInit := os.LookupEnv("LUA_INIT_5_5")
	if !haveInit {
		initName = "=LUA_INIT"
		initVal, haveInit = os.LookupEnv("LUA_INIT")
	}
	if haveInit && initVal != "" && !opts.hasE {
		var initProto *compiler.Proto
		var err error
		if strings.HasPrefix(initVal, "@") {
			fname := initVal[1:]
			src, rerr := os.ReadFile(fname)
			if rerr != nil {
				fmt.Fprintf(stderr, "%s: cannot open %s\n", progName, fname)
				return 1
			}
			initProto, err = compileChunk("@"+fname, fname, string(src))
		} else {
			initProto, err = compileChunk(initName, initName[1:], initVal)
		}
		if err != nil {
			code, _ := loadErr(err)
			return code
		}
		if code, stop := finish(runProto(v, initProto)); stop {
			return code
		}
	}

	// -e and -l, in command-line order (lua.c runargs).
	for _, act := range opts.actions {
		if act.kind == 'e' {
			p, err := compileChunk("=(command line)", "(command line)", act.arg)
			if err != nil {
				code, _ := loadErr(err)
				return code
			}
			if code, stop := finish(runProto(v, p)); stop {
				return code
			}
			continue
		}
		global, mod := act.arg, act.arg
		if i := strings.IndexByte(act.arg, '='); i >= 0 {
			global, mod = act.arg[:i], act.arg[i+1:]
		}
		code, exited, err := guardExit(func() error {
			res, err := v.ProtectedCall(v.GetGlobal("require"), []vm.Value{vm.NewString(mod)})
			if err == nil {
				val := vm.Nil
				if len(res) > 0 {
					val = res[0]
				}
				v.SetGlobal(global, val)
			}
			return err
		})
		if code, stop := finish(code, exited, err); stop {
			return code
		}
	}

	// Run the script, passing its arguments as the main chunk's varargs.
	if scriptName != "" {
		var proto *compiler.Proto
		var err error
		if fromStdin {
			var src []byte
			if src, err = io.ReadAll(cliStdin); err == nil {
				proto, err = compileChunk("=stdin", "stdin", string(src))
			}
		} else {
			var src []byte
			if src, err = os.ReadFile(scriptName); err != nil {
				fmt.Fprintf(stderr, "Error reading file: %v\n", err)
				return 1
			}
			proto, err = compileChunk("@"+scriptName, scriptName, string(src))
		}
		if err != nil {
			code, _ := loadErr(err)
			return code
		}
		rest := argv[opts.script+1:]
		mainArgs := make([]vm.Value, len(rest))
		for i, a := range rest {
			mainArgs[i] = vm.NewString(a)
		}
		code, exited, err := runProtoArgs(v, proto, mainArgs)
		if code, stop := finish(code, exited, err); stop {
			return code
		}
	}

	// No script and nothing else to do (or -i): run standard input. There is
	// no line-by-line REPL; stdin is executed as one chunk named "stdin".
	// Like lua.c's dofile(NULL) in the no-script case, its failure is
	// reported but does not change the exit status.
	if opts.hasI || (scriptName == "" && !opts.hasEStat && !opts.hasV) {
		src, err := io.ReadAll(cliStdin)
		var proto *compiler.Proto
		if err == nil {
			proto, err = compileChunk("=stdin", "stdin", string(src))
		}
		if err != nil {
			code, _ := loadErr(err)
			if opts.hasI {
				return code
			}
			return 0
		}
		code, _, err := runProto(v, proto)
		if err != nil {
			reportLuaError(v, err, progName, stderr)
			if opts.hasI {
				return 1
			}
			return 0
		}
		return code
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
	return val.String()
}

// compileChunk parses and compiles a text chunk, or undumps a precompiled
// binary chunk (LUA_SIGNATURE prefix) like reference luaL_loadfile.
// name carries the '@'/'=' prefix for proto.Source; displayName is used in
// parser error messages.
func compileChunk(name, displayName, source string) (proto *compiler.Proto, err error) {
	// Like luaL_loadfilex, a file source is checked for the binary signature
	// after an optional BOM and an optional '#' first line. Strings (-e and
	// LUA_INIT code) load like luaL_loadbuffer, which only looks at byte 0.
	isFile := strings.HasPrefix(name, "@")
	bin := source
	if isFile {
		bin = strings.TrimPrefix(bin, "\xEF\xBB\xBF")
		if strings.HasPrefix(bin, "#") {
			if i := strings.IndexByte(bin, '\n'); i >= 0 {
				bin = bin[i+1:]
			} else {
				bin = ""
			}
		}
	}
	// Like lua_load, the first byte alone (LUA_SIGNATURE[0]) selects the
	// undumper, so a bad signature reports "bad binary format".
	if strings.HasPrefix(bin, "\x1b") {
		if !isFile {
			// lua.c loads -e and LUA_INIT strings in text mode.
			return nil, errors.New("attempt to load a binary chunk (mode is 't')")
		}
		// The undumper panics on malformed chunks; report as an error.
		defer func() {
			if r := recover(); r != nil {
				proto = nil
				err = fmt.Errorf("%v", r)
			}
		}()
		// Undump uses its name only in error messages, which reference
		// spells without the '@'/'=' prefix.
		proto, _, err = compiler.Undump([]byte(bin), displayName)
		return proto, err
	}
	// Only a file source drops its BOM and '#' first line (luaL_loadfilex);
	// a string is parsed exactly as given, like luaL_loadbuffer.
	if isFile {
		source = strings.TrimPrefix(source, "\xEF\xBB\xBF")
	}
	block, err := parser.Parse(displayName, source, isFile)
	if err != nil {
		return nil, err
	}
	return compiler.Compile(name, block)
}

// runProto runs a compiled chunk, recovering the os.exit sentinel.
// exited reports whether os.exit terminated the run (exitCode is then its
// status, possibly 0).
func runProto(v *vm.VM, proto *compiler.Proto) (int, bool, error) {
	return runProtoArgs(v, proto, nil)
}

func runProtoArgs(v *vm.VM, proto *compiler.Proto, args []vm.Value) (exitCode int, exited bool, err error) {
	return guardExit(func() error {
		_, err := v.RunArgs(proto, args)
		return err
	})
}

// guardExit runs f, recovering the os.exit sentinel.
func guardExit(f func() error) (exitCode int, exited bool, err error) {
	defer func() {
		if r := recover(); r != nil {
			if exitErr, ok := r.(*vm.LuaExitError); ok {
				exitCode = exitErr.Code
				exited = true
				return
			}
			// Re-panic for unexpected panics
			panic(r)
		}
	}()
	err = f()
	return exitCode, exited, err
}
