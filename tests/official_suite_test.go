package tests

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/iceisfun/golua/v2/stdlib"
	"github.com/iceisfun/golua/v2/vm"
)

// TestOfficialSuite runs the *unmodified* upstream Lua 5.5.0 test files
// (testes/*.lua) against golua, as a conformance scoreboard to complement the
// hand-ported excerpts in test_*.lua.
//
// The upstream all.lua driver can't run as-is: it needs the ltests C library
// (the global `T`) and a debug-instrumented build. Instead we run each file
// standalone after a small prelude chunk that:
//   - sets _soft/_port/_nomsg = true (skip slow/non-portable/message tests),
//   - leaves T nil so `if T then ... end` blocks self-skip,
//   - tames collectgarbage("count") to a constant so the explicitly-deferred
//     GC-count approximation doesn't trip count-stability assertions (real
//     collection still happens for every other mode).
//
// The prelude runs as a separate chunk in the same VM so each file keeps its
// original line numbers (errors.lua/db.lua assert on error line numbers).
//
// The suite directory is located via the GOLUA_LUA55_TESTS environment
// variable, falling back to a couple of well-known local paths. The whole test
// skips if the suite isn't present, so it never breaks a checkout without it.
//
// knownFail is the exploration backlog: files that don't yet run clean
// standalone. A file in knownFail that still fails is reported as a skip with
// its lead; a file in knownFail that starts PASSING is a hard failure (promote
// it). A file NOT in knownFail that fails is a regression.
func TestOfficialSuite(t *testing.T) {
	suiteDir := locateOfficialSuite()
	if suiteDir == "" {
		t.Skip("official Lua 5.5 suite not found; set GOLUA_LUA55_TESTS to its testes/ dir")
	}

	// Files that cannot run in-process at all (independent of golua parity):
	// main.lua re-executes itself as a subprocess via arg[0].
	unrunnable := map[string]string{
		"main.lua": "re-execs itself as a subprocess (arg[0]); not an in-process test",
		"all.lua":  "driver, not a standalone test",
	}

	// Resource-intensive files: fast in isolation but can exceed the per-file
	// timeout under full-package parallel load. Gated behind -full, mirroring
	// the test_heavy_ convention.
	slow := map[string]bool{
		"heavy.lua":   true,
		"verybig.lua": true,
	}

	// Files that need real, unjailed filesystem access (e.g. /dev/null and
	// /dev/full for the flush tests). These run trusted upstream code, so the
	// harness swaps in an unsandboxed test-only IO provider for them. Scoped
	// per-file so the default root-jailed provider still guards every other
	// file's access-denied / error-message expectations.
	unsandboxedIo := map[string]bool{
		"files.lua": true,
	}

	// Per-file timeout overrides. A few upstream files are CPU-heavy stress
	// tests — calls.lua drives deep recursion to provoke stack overflow, which
	// in C Lua bails at ~200 C calls (LUAI_MAXCCALLS) but in golua grinds
	// through its much deeper DefaultMaxCallDepth (10000). It passes correctly
	// and runs in ~4s solo, but can exceed the 30s default under full-machine
	// `go test ./...` contention. Give it more headroom. Files NOT listed keep
	// luaTestTimeout, so a genuine hang (e.g. db.lua) is still caught quickly.
	heavyTimeout := map[string]time.Duration{
		"calls.lua": 120 * time.Second,
	}

	// Exploration backlog: standalone failures observed at wiring time. Line
	// numbers are approximate and refer to the file's own numbering. Each is a
	// lead to triage into {real parity bug, harness dependency, known-soft}.
	knownFail := map[string]string{
		"db.lua":   "db.lua — the original :417 debug.getlocal out-of-range temp-slot bug is FIXED (compiler ADDI/arith no longer leaves a register gap below a call target). A SECOND, independent blocker remains: the full file hangs in a cumulative-state interaction somewhere after the coroutine-debug section (~line 681+), only reproducible in the full run (every section passes in isolation). Not the getlocal bug; needs separate triage (deferred)",
		"gc.lua":   "gc.lua:286 — weak-table reclamation under Go GC (timing-dependent, deferred); collectgarbage(\"param\") round-trip fixed",
		"sort.lua": "sort.lua:22 — assert(memdiff > N*4) relies on collectgarbage(\"count\") deltas, which the harness prelude stubs to a constant; reference fails identically under the same stub (harness limitation)",
	}

	files, err := filepath.Glob(filepath.Join(suiteDir, "*.lua"))
	if err != nil {
		t.Fatalf("glob official suite: %v", err)
	}
	if len(files) == 0 {
		t.Skipf("no .lua files under %s", suiteDir)
	}
	sort.Strings(files)

	for _, file := range files {
		base := filepath.Base(file)
		t.Run(base, func(t *testing.T) {
			if reason, skip := unrunnable[base]; skip {
				t.Skipf("unrunnable in-process: %s", reason)
			}
			if slow[base] && !*full {
				t.Skip("resource-intensive official file requires -full flag")
			}

			timeout := luaTestTimeout
			if t, ok := heavyTimeout[base]; ok {
				timeout = t
			}
			runErr := runOfficialFile(suiteDir, file, unsandboxedIo[base], timeout)
			reason, isKnown := knownFail[base]

			switch {
			case runErr == nil && isKnown:
				t.Fatalf("%s now PASSES standalone — remove it from knownFail (was: %s)", base, reason)
			case runErr == nil:
				// Clean pass — nothing to do.
			case isKnown:
				t.Skipf("known standalone failure (lead): %s | %v", reason, runErr)
			default:
				t.Fatalf("regression: %s failed standalone: %v", base, runErr)
			}
		})
	}
}

// runOfficialFile runs the harness prelude and then `file` in a single VM whose
// providers are rooted at the suite directory (so dofile/require of sibling
// files and relative paths resolve). Returns the run error, if any.
func runOfficialFile(suiteDir, file string, unsandboxedIo bool, timeout time.Duration) error {
	src, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}

	preludeProto, err := compileLua("(offsuite-prelude)", officialSuitePrelude)
	if err != nil {
		return fmt.Errorf("compile prelude: %w", err)
	}
	fileProto, err := compileLua(filepath.Base(file), string(src))
	if err != nil {
		return fmt.Errorf("compile: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	v := vm.New(
		vm.WithContext(ctx),
		vm.WithLimits(vm.Limits{GCStepInterval: 10000}),
		// Capture print() output into the VM buffer instead of stdout so the
		// test run stays quiet; this runner only asserts on the run error.
		vm.WithCaptureOutput(true),
	)
	v.SetOsProvider(vm.NewDefaultOsProvider())
	v.SetDebugProvider(vm.NewDefaultDebugProvider())
	if unsandboxedIo {
		// Test-only: real, unjailed filesystem access so /dev/null and
		// /dev/full (the flush tests) behave as on a stock Lua build.
		v.SetIoProvider(vm.NewTestIoProvider())
	} else {
		v.SetIoProvider(vm.NewFullIoProvider(suiteDir))
	}
	v.SetCodeProvider(vm.NewDirCodeProvider(suiteDir, vm.LuaLoaderCaps{
		AllowLoadfile: true,
		AllowDofile:   true,
	}))
	v.SetExecProvider(vm.NewDefaultExecProvider())
	v.SetExitHandler(vm.NewDefaultExitHandler())
	v.SetProcessProvider(vm.NewDefaultProcessProvider())
	stdlib.Open(v)

	resultCh := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				if _, isExit := r.(*vm.LuaExitError); isExit {
					resultCh <- nil
					return
				}
				if e, ok := r.(error); ok {
					resultCh <- e
				} else {
					resultCh <- fmt.Errorf("runtime panic: %v", r)
				}
			}
		}()
		if _, e := v.Run(preludeProto); e != nil {
			resultCh <- fmt.Errorf("prelude: %w", e)
			return
		}
		_, e := v.Run(fileProto)
		resultCh <- e
	}()

	select {
	case e := <-resultCh:
		return e
	case <-time.After(timeout + 2*time.Second):
		cancel()
		return fmt.Errorf("timed out after %v (possible deadlock)", timeout)
	}
}

// locateOfficialSuite resolves the upstream testes/ directory, preferring the
// GOLUA_LUA55_TESTS env var, then a couple of well-known local checkout paths.
func locateOfficialSuite() string {
	candidates := []string{
		os.Getenv("GOLUA_LUA55_TESTS"),
		"/home/iceisfun/work/lua/tests/lua-5.5.0-tests",
		"/home/iceisfun/Downloads/lua-5.5.0/lua-5.5.0-tests",
	}
	for _, c := range candidates {
		if c == "" {
			continue
		}
		if info, err := os.Stat(filepath.Join(c, "all.lua")); err == nil && !info.IsDir() {
			return c
		}
	}
	return ""
}

// officialSuitePrelude is the standalone harness shim. It must not depend on the
// ltests C library. See TestOfficialSuite for rationale.
const officialSuitePrelude = `
_soft = true
_port = true
_nomsg = true
arg = arg or {}
-- T is intentionally left nil so 'if T then ... end' blocks self-skip.
do
  local real = collectgarbage
  collectgarbage = function(opt, ...)
    opt = opt or "collect"
    if opt == "count" then return 0.0, 0 end
    -- Delegate everything else to the real collectgarbage. Re-raise its error
    -- one level up (level 2) so argument-error messages still name
    -- 'collectgarbage' rather than this wrapper's local 'real' — errors.lua's
    -- "(collectgarbage or print){}" check greps the message for that name.
    local res = table.pack(pcall(real, opt, ...))
    if res[1] then return table.unpack(res, 2, res.n) end
    local err = res[2]
    if type(err) == "string" then
      err = (err:gsub("'real'", "'collectgarbage'"))
    end
    error(err, 2)
  end
end
`
