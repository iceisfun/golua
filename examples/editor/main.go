// Example: a browser-based Lua editor with live diagnostics and sandboxed
// execution.
//
// It serves a Monaco-based code editor at http://127.0.0.1:8080 with two
// JSON endpoints:
//
//   - POST /api/check — parses the source with the check package and returns
//     diagnostics positioned for Monaco's marker API (red squiggles as you
//     type).
//   - POST /api/run — compiles and runs the source in a fresh, sandboxed VM
//     with execution limits and a deadline, then returns captured output plus
//     a structured error (with the offending line) if anything went wrong.
//
// Each request gets its own VM, so the handlers are safe to run concurrently.
package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/iceisfun/golua/v2/check"
	"github.com/iceisfun/golua/v2/compiler"
	"github.com/iceisfun/golua/v2/parser"
	"github.com/iceisfun/golua/v2/stdlib"
	"github.com/iceisfun/golua/v2/vm"
)

//go:embed index.html
var content embed.FS

const (
	addr = "127.0.0.1:8080"

	// sourceName is the chunk name used for parse/compile/runtime errors.
	// golua prefixes error messages with "<sourceName>:<line>:", which we
	// parse back out to position runtime errors in the editor.
	sourceName = "editor"

	// execTimeout is the wall-clock budget for a single Run request. It backs
	// up the instruction limit: a tight loop that does no work between
	// checkpoints still cannot run forever.
	execTimeout = 5 * time.Second

	// maxBody caps request bodies to keep a single client from sending us an
	// unbounded amount of source to parse/compile.
	maxBody = 256 << 10 // 256 KiB
)

// sandboxLimits constrains what a submitted script may consume. The VM is also
// sandboxed by omission: no io/os/exec/net providers are registered, so the
// script gets only pure computation plus the safe parts of the standard
// library (string, table, math, etc.) — it cannot touch the filesystem,
// environment, or network.
var sandboxLimits = vm.Limits{
	MaxInstructions: 50_000_000, // ~seconds of CPU; paired with execTimeout
	MaxCallDepth:    500,        // catchable "stack overflow" well before a Go crash
	MaxStackSlots:   100_000,    // bounds a single unpack/deep call chain
}

// errLineRe extracts the 1-based line number from a golua error message. It
// accepts both the parse/compile prefix ("editor:<line>:") and the runtime
// short-source prefix ('[string "editor"]:<line>:'), matching anywhere in the
// string so it works for every error phase.
var errLineRe = regexp.MustCompile(`(?:\[string ")?` + regexp.QuoteMeta(sourceName) + `(?:"\])?:(\d+):`)

func main() {
	http.Handle("GET /", http.FileServer(http.FS(content)))
	http.HandleFunc("POST /api/check", handleCheck)
	http.HandleFunc("POST /api/run", handleRun)

	fmt.Printf("GoLua Editor: http://%s\n", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

// --- /api/check: live diagnostics -------------------------------------------

type checkRequest struct {
	Source string `json:"source"`
}

type checkResponse struct {
	Diagnostics []check.Diagnostic `json:"diagnostics"`
}

func handleCheck(w http.ResponseWriter, r *http.Request) {
	var req checkRequest
	if !decodeBody(w, r, &req) {
		return
	}

	// Check parses partial/incomplete source and never returns a nil result,
	// so callers can render diagnostics on every keystroke without guarding.
	result := check.Check(sourceName, req.Source)

	// Diagnostic fields already match Monaco's IMarkerData (1-based positions,
	// severity values matching MarkerSeverity), so they serialize straight to
	// the client.
	writeJSON(w, checkResponse{Diagnostics: result.Diagnostics})
}

// --- /api/run: sandboxed execution ------------------------------------------

type runRequest struct {
	Source string `json:"source"`
}

type runResponse struct {
	Output string `json:"output"`
	// Error is empty on success. On failure it carries a human-readable,
	// phase-tagged message ("parse error: ...", "runtime error: ...", etc.).
	Error string `json:"error,omitempty"`
	// ErrorLine is the 1-based source line the error points at, or 0 if the
	// error has no associated line (e.g. error(table)). The frontend uses it
	// to place a marker on the offending line.
	ErrorLine int `json:"errorLine,omitempty"`
	// TimedOut is true when execution hit the deadline rather than failing.
	TimedOut bool `json:"timedOut,omitempty"`
	// DurationMs is the wall-clock time spent compiling and running.
	DurationMs int64 `json:"durationMs"`
}

func handleRun(w http.ResponseWriter, r *http.Request) {
	var req runRequest
	if !decodeBody(w, r, &req) {
		return
	}

	start := time.Now()
	// Derive from the request context so that a client disconnect cancels the
	// running script promptly, in addition to the execTimeout deadline.
	resp := runScript(r.Context(), req.Source)
	resp.DurationMs = time.Since(start).Milliseconds()

	writeJSON(w, resp)
}

// runScript compiles and executes source in a fresh, sandboxed VM. It always
// returns whatever output was captured before an error (so a script that
// prints then fails still shows its output).
func runScript(ctx context.Context, source string) runResponse {
	block, err := parser.Parse(sourceName, source)
	if err != nil {
		return runResponse{Error: "parse error: " + cleanMsg(err), ErrorLine: errorLine(err)}
	}

	proto, err := compiler.Compile(sourceName, block)
	if err != nil {
		return runResponse{Error: "compile error: " + cleanMsg(err), ErrorLine: errorLine(err)}
	}

	runCtx, cancel := context.WithTimeout(ctx, execTimeout)
	defer cancel()

	v := vm.New(
		vm.WithContext(runCtx),
		vm.WithLimits(sandboxLimits),
		vm.WithCaptureOutput(true), // route print() into a buffer, not stdout
	)
	stdlib.Open(v)
	defer v.Close(context.Background()) // release providers / finalizers

	// v.Run can return an error or, for some host-level conditions, panic;
	// recover() converts a panic into an ordinary error so a single bad script
	// can never take down the server.
	var runErr error
	func() {
		defer func() {
			if rec := recover(); rec != nil {
				runErr = fmt.Errorf("%v", rec)
			}
		}()
		_, runErr = v.Run(proto)
	}()

	resp := runResponse{Output: strings.Join(v.OutputLines(), "\n")}
	if runErr == nil {
		return resp
	}

	switch {
	case runCtx.Err() == context.DeadlineExceeded:
		resp.TimedOut = true
		resp.Error = fmt.Sprintf("execution timed out after %s", execTimeout)
	case errors.Is(runCtx.Err(), context.Canceled):
		resp.Error = "execution canceled"
	default:
		resp.Error = "runtime error: " + cleanMsg(runErr)
		resp.ErrorLine = errorLine(runErr)
	}
	return resp
}

// --- helpers ----------------------------------------------------------------

// cleanMsg normalizes the Lua short-source label ('[string "editor"]') to the
// bare chunk name so messages read "editor:<line>: ..." across every phase.
func cleanMsg(err error) string {
	return strings.ReplaceAll(err.Error(), `[string "`+sourceName+`"]`, sourceName)
}

// errorLine pulls the 1-based source line out of a golua error message, or
// returns 0 if the error carries no position.
func errorLine(err error) int {
	if err == nil {
		return 0
	}
	m := errLineRe.FindStringSubmatch(err.Error())
	if m == nil {
		return 0
	}
	n := 0
	for _, c := range m[1] {
		n = n*10 + int(c-'0')
	}
	return n
}

// decodeBody reads a size-capped JSON body into dst, writing a 400 and
// returning false on failure.
func decodeBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
