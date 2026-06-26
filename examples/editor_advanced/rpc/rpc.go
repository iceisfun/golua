// Package rpc implements a minimal JSON-RPC 2.0 router over HTTP.
package rpc

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// Standard JSON-RPC 2.0 error codes.
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)

// Request is a JSON-RPC 2.0 request.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// Response is a JSON-RPC 2.0 response.
type Response struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id"`
	Result  any    `json:"result,omitempty"`
	Error   *Error `json:"error,omitempty"`
}

// Error is a JSON-RPC 2.0 error object.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Handler processes a JSON-RPC method call.
type Handler func(params json.RawMessage) (any, error)

// Router dispatches JSON-RPC 2.0 requests to registered handlers.
type Router struct {
	mu       sync.RWMutex
	handlers map[string]Handler
}

// NewRouter creates a new JSON-RPC router.
func NewRouter() *Router {
	return &Router{handlers: make(map[string]Handler)}
}

// Register adds a method handler.
func (r *Router) Register(method string, h Handler) {
	r.mu.Lock()
	r.handlers[method] = h
	r.mu.Unlock()
}

// HandleHTTP returns an http.HandlerFunc that processes JSON-RPC requests.
func (r *Router) HandleHTTP() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var rpcReq Request
		if err := json.NewDecoder(http.MaxBytesReader(w, req.Body, 64<<10)).Decode(&rpcReq); err != nil {
			json.NewEncoder(w).Encode(Response{
				JSONRPC: "2.0",
				Error:   &Error{Code: ParseError, Message: "parse error"},
			})
			return
		}

		if rpcReq.JSONRPC != "2.0" || rpcReq.Method == "" {
			json.NewEncoder(w).Encode(Response{
				JSONRPC: "2.0",
				ID:      rpcReq.ID,
				Error:   &Error{Code: InvalidRequest, Message: "invalid request"},
			})
			return
		}

		r.mu.RLock()
		h, ok := r.handlers[rpcReq.Method]
		r.mu.RUnlock()

		if !ok {
			json.NewEncoder(w).Encode(Response{
				JSONRPC: "2.0",
				ID:      rpcReq.ID,
				Error:   &Error{Code: MethodNotFound, Message: "method not found: " + rpcReq.Method},
			})
			return
		}

		result, err := call(h, rpcReq.Params)
		if err != nil {
			code := InternalError
			if rpcErr, ok := err.(*Error); ok {
				json.NewEncoder(w).Encode(Response{
					JSONRPC: "2.0",
					ID:      rpcReq.ID,
					Error:   rpcErr,
				})
				return
			}
			json.NewEncoder(w).Encode(Response{
				JSONRPC: "2.0",
				ID:      rpcReq.ID,
				Error:   &Error{Code: code, Message: err.Error()},
			})
			return
		}

		json.NewEncoder(w).Encode(Response{
			JSONRPC: "2.0",
			ID:      rpcReq.ID,
			Result:  result,
		})
	}
}

func (e *Error) Error() string { return e.Message }

// call invokes a handler, converting any panic into a JSON-RPC InternalError so
// a single buggy method can't tear down the HTTP connection or the server.
func call(h Handler, params json.RawMessage) (result any, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = &Error{Code: InternalError, Message: fmt.Sprintf("handler panic: %v", r)}
		}
	}()
	return h(params)
}
