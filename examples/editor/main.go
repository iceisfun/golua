// Example: Monaco editor with live Lua diagnostics and execution.
//
// Serves a browser-based code editor at http://127.0.0.1:8080 that
// provides real-time syntax checking via the check package and a Run
// button that executes Lua in a sandboxed VM.
package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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

type checkRequest struct {
	Source string `json:"source"`
}

type checkResponse struct {
	Diagnostics []check.Diagnostic `json:"diagnostics"`
}

type runRequest struct {
	Source string `json:"source"`
}

type runResponse struct {
	Output string `json:"output"`
	Error  string `json:"error,omitempty"`
}

func main() {
	http.Handle("GET /", http.FileServer(http.FS(content)))
	http.HandleFunc("POST /api/check", handleCheck)
	http.HandleFunc("POST /api/run", handleRun)

	addr := "127.0.0.1:8080"
	fmt.Printf("GoLua Editor: http://%s\n", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func handleCheck(w http.ResponseWriter, r *http.Request) {
	var req checkRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	result := check.Check("editor", req.Source)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(checkResponse{Diagnostics: result.Diagnostics})
}

func handleRun(w http.ResponseWriter, r *http.Request) {
	var req runRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	output, runErr := executeLua(req.Source)

	resp := runResponse{Output: output}
	if runErr != nil {
		resp.Error = runErr.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func executeLua(source string) (output string, retErr error) {
	block, err := parser.Parse("editor", source)
	if err != nil {
		return "", fmt.Errorf("parse error: %w", err)
	}

	proto, err := compiler.Compile("editor", block)
	if err != nil {
		return "", fmt.Errorf("compile error: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	v := vm.New(
		vm.WithContext(ctx),
		vm.WithLimits(vm.Limits{
			MaxInstructions: 10_000_000,
			MaxCallDepth:    200,
			MaxStackSlots:   10_000,
		}),
		vm.WithCaptureOutput(true),
	)
	stdlib.Open(v)

	defer func() {
		if r := recover(); r != nil {
			output = strings.Join(v.OutputLines(), "\n")
			retErr = fmt.Errorf("runtime error: %v", r)
		}
	}()

	if _, err := v.Run(proto); err != nil {
		return strings.Join(v.OutputLines(), "\n"), fmt.Errorf("runtime error: %v", err)
	}

	return strings.Join(v.OutputLines(), "\n"), nil
}
