// Package http implements an optional HTTP module for GoLua.
//
// This module is NOT part of standard Lua. It must be explicitly enabled by
// the host application:
//
//	stdlib.Open(v)
//	http.Open(v)
//
// The module provides http.get, http.post, http.put, http.patch, http.delete,
// http.head, http.options, and http.fetch for full request control.
//
// All requests use the VM context for cancellation and enforce a default
// 1-minute timeout. Lua tables passed as request bodies are automatically
// encoded as JSON.
package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/iceisfun/golua/vm"
)

const defaultTimeout = 60 * time.Second

// Open registers the http module as a global in the VM.
func Open(v *vm.VM) {
	httpTable := vm.NewEmptyTable()
	httpTable.SetString("get", vm.NewNativeFunc(httpGet))
	httpTable.SetString("post", vm.NewNativeFunc(httpPost))
	httpTable.SetString("put", vm.NewNativeFunc(httpPut))
	httpTable.SetString("patch", vm.NewNativeFunc(httpPatch))
	httpTable.SetString("delete", vm.NewNativeFunc(httpDelete))
	httpTable.SetString("head", vm.NewNativeFunc(httpHead))
	httpTable.SetString("options", vm.NewNativeFunc(httpOptions))
	httpTable.SetString("fetch", vm.NewNativeFunc(httpFetch))
	v.SetGlobal("http", vm.NewTable(httpTable))
}

// httpGet implements http.get(url [, options])
func httpGet(v *vm.VM) int {
	url := getString(v, 1, "http.get")
	opts := getOptions(v, 2)
	opts.Method = "GET"
	opts.URL = url
	return doRequest(v, opts)
}

// httpPost implements http.post(url, body [, options])
func httpPost(v *vm.VM) int {
	url := getString(v, 1, "http.post")
	body := v.Get(2)
	opts := getOptions(v, 3)
	opts.Method = "POST"
	opts.URL = url
	opts.Body = body
	if opts.ContentType == "" && body.IsTable() {
		opts.ContentType = "application/json"
	}
	return doRequest(v, opts)
}

// httpPut implements http.put(url, body [, options])
func httpPut(v *vm.VM) int {
	url := getString(v, 1, "http.put")
	body := v.Get(2)
	opts := getOptions(v, 3)
	opts.Method = "PUT"
	opts.URL = url
	opts.Body = body
	if opts.ContentType == "" && body.IsTable() {
		opts.ContentType = "application/json"
	}
	return doRequest(v, opts)
}

// httpPatch implements http.patch(url, body [, options])
func httpPatch(v *vm.VM) int {
	url := getString(v, 1, "http.patch")
	body := v.Get(2)
	opts := getOptions(v, 3)
	opts.Method = "PATCH"
	opts.URL = url
	opts.Body = body
	if opts.ContentType == "" && body.IsTable() {
		opts.ContentType = "application/json"
	}
	return doRequest(v, opts)
}

// httpDelete implements http.delete(url [, options])
func httpDelete(v *vm.VM) int {
	url := getString(v, 1, "http.delete")
	opts := getOptions(v, 2)
	opts.Method = "DELETE"
	opts.URL = url
	return doRequest(v, opts)
}

// httpHead implements http.head(url [, options])
func httpHead(v *vm.VM) int {
	url := getString(v, 1, "http.head")
	opts := getOptions(v, 2)
	opts.Method = "HEAD"
	opts.URL = url
	return doRequest(v, opts)
}

// httpOptions implements http.options(url [, options])
func httpOptions(v *vm.VM) int {
	url := getString(v, 1, "http.options")
	opts := getOptions(v, 2)
	opts.Method = "OPTIONS"
	opts.URL = url
	return doRequest(v, opts)
}

// httpFetch implements http.fetch(options) for full request control.
func httpFetch(v *vm.VM) int {
	arg := v.Get(1)
	if !arg.IsTable() {
		panic("bad argument #1 to 'http.fetch' (table expected, got " + arg.Type() + ")")
	}
	tbl := arg.AsTable()

	opts := requestOpts{}

	if urlVal := tbl.Get(vm.NewString("url")); urlVal.IsString() {
		opts.URL = urlVal.AsString()
	} else {
		panic("bad argument #1 to 'http.fetch' (field 'url' is required)")
	}

	if methodVal := tbl.Get(vm.NewString("method")); methodVal.IsString() {
		opts.Method = strings.ToUpper(methodVal.AsString())
	} else {
		opts.Method = "GET"
	}

	if headersVal := tbl.Get(vm.NewString("headers")); headersVal.IsTable() {
		opts.Headers = headersVal.AsTable()
	}

	if bodyVal := tbl.Get(vm.NewString("body")); !bodyVal.IsNil() {
		opts.Body = bodyVal
		if opts.ContentType == "" && bodyVal.IsTable() {
			opts.ContentType = "application/json"
		}
	}

	if timeoutVal := tbl.Get(vm.NewString("timeout")); timeoutVal.IsNumber() {
		if timeoutVal.IsInt() {
			opts.Timeout = time.Duration(timeoutVal.AsInt()) * time.Second
		} else {
			opts.Timeout = time.Duration(timeoutVal.AsFloat() * float64(time.Second))
		}
	}

	return doRequest(v, opts)
}

type requestOpts struct {
	Method      string
	URL         string
	Headers     vm.LuaTable
	Body        vm.Value
	Timeout     time.Duration
	ContentType string
}

func getString(v *vm.VM, idx int, fname string) string {
	val := v.Get(idx)
	if val.IsString() {
		return val.AsString()
	}
	panic(fmt.Sprintf("bad argument #%d to '%s' (string expected, got %s)", idx, fname, val.Type()))
}

func getOptions(v *vm.VM, idx int) requestOpts {
	opts := requestOpts{}
	arg := v.Get(idx)
	if arg.IsNil() {
		return opts
	}
	if !arg.IsTable() {
		return opts
	}
	tbl := arg.AsTable()

	if headersVal := tbl.Get(vm.NewString("headers")); headersVal.IsTable() {
		opts.Headers = headersVal.AsTable()
	}
	if timeoutVal := tbl.Get(vm.NewString("timeout")); timeoutVal.IsNumber() {
		if timeoutVal.IsInt() {
			opts.Timeout = time.Duration(timeoutVal.AsInt()) * time.Second
		} else {
			opts.Timeout = time.Duration(timeoutVal.AsFloat() * float64(time.Second))
		}
	}
	return opts
}

// luaTableToJSON converts a Lua table to a JSON byte slice.
func luaTableToJSON(tbl vm.LuaTable) ([]byte, error) {
	return json.Marshal(luaTableToInterface(tbl))
}

// luaTableToInterface converts a Lua table to a Go interface{} suitable for
// json.Marshal. Sequential integer keys starting from 1 produce an array;
// otherwise a map is produced.
func luaTableToInterface(tbl vm.LuaTable) any {
	// Check if it's an array (sequential integer keys starting from 1)
	length := tbl.Len()
	if length > 0 {
		isArray := true
		for i := 1; i <= length; i++ {
			if tbl.Get(vm.NewInt(int64(i))).IsNil() {
				isArray = false
				break
			}
		}
		if isArray {
			// Check there are no keys beyond the sequence
			hasExtra := false
			key := vm.Nil
			for {
				nextKey, _, err := tbl.Next(key)
				if err != nil || nextKey.IsNil() {
					break
				}
				if nextKey.IsInt() {
					idx := nextKey.AsInt()
					if idx >= 1 && idx <= int64(length) {
						key = nextKey
						continue
					}
				}
				hasExtra = true
				break
			}
			if !hasExtra {
				arr := make([]any, length)
				for i := 1; i <= length; i++ {
					arr[i-1] = luaValueToInterface(tbl.Get(vm.NewInt(int64(i))))
				}
				return arr
			}
		}
	}

	// Object
	m := make(map[string]any)
	key := vm.Nil
	for {
		nextKey, val, err := tbl.Next(key)
		if err != nil || nextKey.IsNil() {
			break
		}
		var keyStr string
		if nextKey.IsString() {
			keyStr = nextKey.AsString()
		} else if nextKey.IsInt() {
			keyStr = fmt.Sprintf("%d", nextKey.AsInt())
		} else if nextKey.IsFloat() {
			keyStr = fmt.Sprintf("%g", nextKey.AsFloat())
		} else {
			keyStr = fmt.Sprintf("%v", nextKey)
		}
		m[keyStr] = luaValueToInterface(val)
		key = nextKey
	}
	return m
}

func luaValueToInterface(v vm.Value) any {
	switch {
	case v.IsNil():
		return nil
	case v.IsBool():
		return v.AsBool()
	case v.IsInt():
		return v.AsInt()
	case v.IsFloat():
		return v.AsFloat()
	case v.IsString():
		return v.AsString()
	case v.IsTable():
		return luaTableToInterface(v.AsTable())
	default:
		return v.String()
	}
}

// jsonToLuaValue converts a Go value (from json.Unmarshal) to a Lua Value.
func jsonToLuaValue(val any) vm.Value {
	switch v := val.(type) {
	case nil:
		return vm.Nil
	case bool:
		return vm.NewBool(v)
	case float64:
		// JSON numbers are float64; convert to int if it's a whole number
		if v == float64(int64(v)) && v >= -9007199254740992 && v <= 9007199254740992 {
			return vm.NewInt(int64(v))
		}
		return vm.NewFloat(v)
	case string:
		return vm.NewString(v)
	case []any:
		tbl := vm.NewEmptyTable()
		for i, elem := range v {
			tbl.Set(vm.NewInt(int64(i+1)), jsonToLuaValue(elem))
		}
		return vm.NewTable(tbl)
	case map[string]any:
		tbl := vm.NewEmptyTable()
		for key, elem := range v {
			tbl.SetString(key, jsonToLuaValue(elem))
		}
		return vm.NewTable(tbl)
	default:
		return vm.NewString(fmt.Sprintf("%v", v))
	}
}

func doRequest(v *vm.VM, opts requestOpts) int {
	// Build request body
	var bodyReader io.Reader
	if !opts.Body.IsNil() {
		if opts.Body.IsString() {
			bodyReader = strings.NewReader(opts.Body.AsString())
		} else if opts.Body.IsTable() {
			data, err := luaTableToJSON(opts.Body.AsTable())
			if err != nil {
				panic(fmt.Sprintf("http: failed to encode body as JSON: %s", err.Error()))
			}
			bodyReader = strings.NewReader(string(data))
		} else {
			bodyReader = strings.NewReader(opts.Body.String())
		}
	}

	// Build context with timeout
	timeout := defaultTimeout
	if opts.Timeout > 0 {
		timeout = opts.Timeout
	}
	baseCtx := v.Context()
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	ctx, cancel := context.WithTimeout(baseCtx, timeout)
	defer cancel()

	// Build request
	req, err := http.NewRequestWithContext(ctx, opts.Method, opts.URL, bodyReader)
	if err != nil {
		panic(fmt.Sprintf("http: %s", err.Error()))
	}

	// Set content type for table bodies
	if opts.ContentType != "" {
		req.Header.Set("Content-Type", opts.ContentType)
	}

	// Set custom headers
	if opts.Headers != nil {
		key := vm.Nil
		for {
			nextKey, val, err := opts.Headers.Next(key)
			if err != nil || nextKey.IsNil() {
				break
			}
			if nextKey.IsString() && val.IsString() {
				req.Header.Set(nextKey.AsString(), val.AsString())
			}
			key = nextKey
		}
	}

	// Execute request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(fmt.Sprintf("http: %s", err.Error()))
	}
	defer resp.Body.Close()

	// Read body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(fmt.Sprintf("http: failed to read response body: %s", err.Error()))
	}

	// Build response table
	result := vm.NewEmptyTable()
	result.SetString("status", vm.NewInt(int64(resp.StatusCode)))
	result.SetString("status_text", vm.NewString(resp.Status))
	result.SetString("body", vm.NewString(string(bodyBytes)))

	// Response headers
	headerTable := vm.NewEmptyTable()
	for name, values := range resp.Header {
		if len(values) > 0 {
			headerTable.SetString(name, vm.NewString(values[0]))
		}
	}
	result.SetString("headers", vm.NewTable(headerTable))

	// json() convenience method
	bodyStr := string(bodyBytes)
	result.SetString("json", vm.NewNativeFunc(func(v *vm.VM) int {
		var parsed any
		if err := json.Unmarshal([]byte(bodyStr), &parsed); err != nil {
			panic(fmt.Sprintf("http: failed to parse JSON response: %s", err.Error()))
		}
		v.Set(0, jsonToLuaValue(parsed))
		return 1
	}))

	v.Set(0, vm.NewTable(result))
	return 1
}
