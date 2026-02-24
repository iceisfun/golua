// Example: Monaco editor with live Lua diagnostics, completion, hover, and execution.
//
// Serves a browser-based IDE at http://127.0.0.1:8080 with JSON-RPC 2.0
// language services (completion, hover, diagnostics) and a Run endpoint.
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

	"github.com/iceisfun/golua/check"
	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/examples/editor_advanced/language"
	"github.com/iceisfun/golua/examples/editor_advanced/rpc"
	"github.com/iceisfun/golua/examples/editor_advanced/session"
	"github.com/iceisfun/golua/parser"
	"github.com/iceisfun/golua/stdlib"
	"github.com/iceisfun/golua/vm"
)

//go:embed index.html
var content embed.FS

func main() {
	mgr := session.NewManager()
	router := rpc.NewRouter()

	registerMethods(router, mgr)

	http.Handle("GET /", http.FileServer(http.FS(content)))
	http.HandleFunc("POST /api/rpc", router.HandleHTTP())
	http.HandleFunc("POST /api/run", handleRun)

	addr := "127.0.0.1:8080"
	fmt.Printf("GoLua Editor (Advanced): http://%s\n", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

// --- JSON-RPC methods ---

func registerMethods(router *rpc.Router, mgr *session.Manager) {
	router.Register("initialize", func(params json.RawMessage) (any, error) {
		return map[string]any{
			"capabilities": map[string]any{
				"completionProvider":  true,
				"hoverProvider":       true,
				"diagnosticsProvider": true,
			},
		}, nil
	})

	router.Register("shutdown", func(params json.RawMessage) (any, error) {
		return nil, nil
	})

	router.Register("textDocument/didOpen", func(params json.RawMessage) (any, error) {
		var p struct {
			URI     string `json:"uri"`
			Version int    `json:"version"`
			Text    string `json:"text"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &rpc.Error{Code: rpc.InvalidParams, Message: "invalid params"}
		}
		mgr.Open(p.URI, p.Version, p.Text)
		return diagnosticsFor(p.Text), nil
	})

	router.Register("textDocument/didChange", func(params json.RawMessage) (any, error) {
		var p struct {
			URI     string `json:"uri"`
			Version int    `json:"version"`
			Text    string `json:"text"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &rpc.Error{Code: rpc.InvalidParams, Message: "invalid params"}
		}
		mgr.Update(p.URI, p.Version, p.Text)
		return diagnosticsFor(p.Text), nil
	})

	router.Register("textDocument/completion", func(params json.RawMessage) (any, error) {
		var p struct {
			URI      string `json:"uri"`
			Line     int    `json:"line"`
			Col      int    `json:"col"`
			LineText string `json:"lineText"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &rpc.Error{Code: rpc.InvalidParams, Message: "invalid params"}
		}
		doc := mgr.Get(p.URI)
		var symbols *language.SymbolTable
		if doc != nil {
			symbols = doc.Symbols
		}
		items := language.Complete(symbols, p.Line, p.Col, p.LineText)
		return map[string]any{"items": items}, nil
	})

	router.Register("textDocument/hover", func(params json.RawMessage) (any, error) {
		var p struct {
			URI  string `json:"uri"`
			Line int    `json:"line"`
			Col  int    `json:"col"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &rpc.Error{Code: rpc.InvalidParams, Message: "invalid params"}
		}
		doc := mgr.Get(p.URI)
		if doc == nil {
			return nil, nil
		}
		result := language.Hover(doc.Symbols, p.Line, p.Col, doc.Text)
		if result == nil {
			return nil, nil
		}
		return result, nil
	})
}

func diagnosticsFor(text string) map[string]any {
	result := check.Check("editor", text)
	return map[string]any{
		"diagnostics": result.Diagnostics,
	}
}

// --- Run endpoint (same as basic editor) ---

type runRequest struct {
	Source string `json:"source"`
}

type runResponse struct {
	Output string `json:"output"`
	Error  string `json:"error,omitempty"`
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
