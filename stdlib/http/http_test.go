package http

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/iceisfun/golua/compiler"
	"github.com/iceisfun/golua/parser"
	"github.com/iceisfun/golua/vm"
)

// newTestVM creates a VM with the http module and basic builtins registered.
func newTestVM() *vm.VM {
	v := vm.New()
	Open(v)
	registerBasics(v)
	return v
}

// runLua compiles and runs a Lua script, returning the VM.
func runLua(t *testing.T, v *vm.VM, code string) {
	t.Helper()
	block, err := parser.Parse("test", code)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto, err := compiler.Compile("test", block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	if _, err := v.Run(proto); err != nil {
		t.Fatalf("execute error: %v", err)
	}
}

func TestGetRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.WriteHeader(200)
		w.Write([]byte("hello world"))
	}))
	defer srv.Close()

	v := newTestVM()
	v.SetGlobal("test_url", vm.NewString(srv.URL))

	runLua(t, v, `
		local res = http.get(test_url)
		assert(res.status == 200, "status should be 200, got " .. tostring(res.status))
		assert(res.body == "hello world", "body mismatch: " .. tostring(res.body))
	`)
}

func TestGetRequestStatusCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte("not found"))
	}))
	defer srv.Close()

	v := newTestVM()
	v.SetGlobal("test_url", vm.NewString(srv.URL))

	runLua(t, v, `
		local res = http.get(test_url)
		assert(res.status == 404, "status should be 404")
		assert(res.body == "not found", "body mismatch")
	`)
}

func TestPostJSONCoercion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		ct := r.Header.Get("Content-Type")
		if ct != "application/json" {
			t.Errorf("expected application/json content-type, got %s", ct)
		}
		body, _ := io.ReadAll(r.Body)
		var data map[string]any
		if err := json.Unmarshal(body, &data); err != nil {
			t.Errorf("failed to parse JSON body: %v", err)
		}
		if data["name"] != "alice" {
			t.Errorf("expected name=alice, got %v", data["name"])
		}
		w.WriteHeader(201)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	v := newTestVM()
	v.SetGlobal("test_url", vm.NewString(srv.URL))

	runLua(t, v, `
		local res = http.post(test_url, { name = "alice", id = 42 })
		assert(res.status == 201, "status should be 201")
	`)
}

func TestPostStringBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if string(body) != "raw body data" {
			t.Errorf("expected raw body, got %s", body)
		}
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	v := newTestVM()
	v.SetGlobal("test_url", vm.NewString(srv.URL))

	runLua(t, v, `
		local res = http.post(test_url, "raw body data")
		assert(res.status == 200)
	`)
}

func TestPostHeaderPropagation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer mytoken" {
			t.Errorf("expected Bearer mytoken, got %s", auth)
		}
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	v := newTestVM()
	v.SetGlobal("test_url", vm.NewString(srv.URL))

	runLua(t, v, `
		local res = http.post(test_url, "data", {
			headers = { Authorization = "Bearer mytoken" }
		})
		assert(res.status == 200)
	`)
}

func TestFetchFullPipeline(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		ua := r.Header.Get("User-Agent")
		if ua != "GoLua" {
			t.Errorf("expected User-Agent GoLua, got %s", ua)
		}
		body, _ := io.ReadAll(r.Body)
		var data map[string]any
		json.Unmarshal(body, &data)
		w.Header().Set("X-Custom", "value")
		w.WriteHeader(200)
		w.Write([]byte(`{"result":"ok"}`))
	}))
	defer srv.Close()

	v := newTestVM()
	v.SetGlobal("test_url", vm.NewString(srv.URL))

	runLua(t, v, `
		local res = http.fetch({
			method = "PUT",
			url = test_url,
			headers = {
				["User-Agent"] = "GoLua"
			},
			body = { id = 123 }
		})
		assert(res.status == 200)
	`)
}

func TestFetchDefaultMethod(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	v := newTestVM()
	v.SetGlobal("test_url", vm.NewString(srv.URL))

	runLua(t, v, `
		local res = http.fetch({ url = test_url })
		assert(res.status == 200)
	`)
}

func TestResponseHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom-Header", "test-value")
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	v := newTestVM()
	v.SetGlobal("test_url", vm.NewString(srv.URL))

	runLua(t, v, `
		local res = http.get(test_url)
		assert(res.headers["X-Custom-Header"] == "test-value",
			"header mismatch: " .. tostring(res.headers["X-Custom-Header"]))
	`)
}

func TestResponseJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"name":"bob","age":30,"active":true}`))
	}))
	defer srv.Close()

	v := newTestVM()
	v.SetGlobal("test_url", vm.NewString(srv.URL))

	runLua(t, v, `
		local res = http.get(test_url)
		local data = res.json()
		assert(data.name == "bob", "name mismatch")
		assert(data.age == 30, "age mismatch")
		assert(data.active == true, "active mismatch")
	`)
}

func TestTimeoutOverride(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(3 * time.Second)
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	v := newTestVM()
	v.SetGlobal("test_url", vm.NewString(srv.URL))
	registerBasics(v)

	runLua(t, v, `
		local ok, err = pcall(http.get, test_url, { timeout = 1 })
		assert(not ok, "should have timed out")
	`)
}

func TestConnectionFailure(t *testing.T) {
	v := newTestVM()
	registerBasics(v)

	v.SetGlobal("test_url", vm.NewString("http://127.0.0.1:1"))

	runLua(t, v, `
		local ok, err = pcall(http.get, test_url)
		assert(not ok, "should have failed to connect")
	`)
}

func TestInvalidURL(t *testing.T) {
	v := newTestVM()
	registerBasics(v)

	runLua(t, v, `
		local ok, err = pcall(http.get, "://invalid")
		assert(not ok, "should have failed with invalid URL")
	`)
}

func TestCustomHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-First") != "one" || r.Header.Get("X-Second") != "two" {
			t.Errorf("missing custom headers")
		}
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	v := newTestVM()
	v.SetGlobal("test_url", vm.NewString(srv.URL))

	runLua(t, v, `
		local res = http.fetch({
			url = test_url,
			headers = {
				["X-First"] = "one",
				["X-Second"] = "two"
			}
		})
		assert(res.status == 200)
	`)
}

func TestPutRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		w.WriteHeader(200)
		w.Write([]byte("updated"))
	}))
	defer srv.Close()

	v := newTestVM()
	v.SetGlobal("test_url", vm.NewString(srv.URL))

	runLua(t, v, `
		local res = http.put(test_url, { key = "value" })
		assert(res.status == 200)
		assert(res.body == "updated")
	`)
}

func TestPatchRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		w.WriteHeader(200)
		w.Write([]byte("patched"))
	}))
	defer srv.Close()

	v := newTestVM()
	v.SetGlobal("test_url", vm.NewString(srv.URL))

	runLua(t, v, `
		local res = http.patch(test_url, { field = "new" })
		assert(res.status == 200)
		assert(res.body == "patched")
	`)
}

func TestDeleteRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(204)
	}))
	defer srv.Close()

	v := newTestVM()
	v.SetGlobal("test_url", vm.NewString(srv.URL))

	runLua(t, v, `
		local res = http.delete(test_url)
		assert(res.status == 204)
	`)
}

func TestHeadRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "HEAD" {
			t.Errorf("expected HEAD, got %s", r.Method)
		}
		w.Header().Set("X-Info", "metadata")
		w.WriteHeader(200)
	}))
	defer srv.Close()

	v := newTestVM()
	v.SetGlobal("test_url", vm.NewString(srv.URL))

	runLua(t, v, `
		local res = http.head(test_url)
		assert(res.status == 200)
		assert(res.headers["X-Info"] == "metadata")
	`)
}

func TestOptionsRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "OPTIONS" {
			t.Errorf("expected OPTIONS, got %s", r.Method)
		}
		w.Header().Set("Allow", "GET, POST, OPTIONS")
		w.WriteHeader(204)
	}))
	defer srv.Close()

	v := newTestVM()
	v.SetGlobal("test_url", vm.NewString(srv.URL))

	runLua(t, v, `
		local res = http.options(test_url)
		assert(res.status == 204)
		assert(res.headers["Allow"] == "GET, POST, OPTIONS")
	`)
}

func TestJSONArrayBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var data []any
		if err := json.Unmarshal(body, &data); err != nil {
			t.Errorf("expected JSON array: %v", err)
		}
		if len(data) != 3 {
			t.Errorf("expected 3 elements, got %d", len(data))
		}
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	v := newTestVM()
	v.SetGlobal("test_url", vm.NewString(srv.URL))

	runLua(t, v, `
		local res = http.post(test_url, { "a", "b", "c" })
		assert(res.status == 200)
	`)
}

func TestJSONResponseArray(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`[1, 2, 3]`))
	}))
	defer srv.Close()

	v := newTestVM()
	v.SetGlobal("test_url", vm.NewString(srv.URL))

	runLua(t, v, `
		local res = http.get(test_url)
		local data = res.json()
		assert(data[1] == 1)
		assert(data[2] == 2)
		assert(data[3] == 3)
	`)
}

func TestFetchMissingURL(t *testing.T) {
	v := newTestVM()
	registerBasics(v)

	runLua(t, v, `
		local ok, err = pcall(http.fetch, {})
		assert(not ok, "should fail without url")
	`)
}

func TestFetchBadArgument(t *testing.T) {
	v := newTestVM()
	registerBasics(v)

	runLua(t, v, `
		local ok, err = pcall(http.fetch, "not a table")
		assert(not ok, "should fail with non-table argument")
	`)
}

func TestStatusText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	v := newTestVM()
	v.SetGlobal("test_url", vm.NewString(srv.URL))

	runLua(t, v, `
		local res = http.get(test_url)
		assert(type(res.status_text) == "string")
	`)
}

// registerBasics adds pcall, assert, type, tostring needed by error-handling tests.
func registerBasics(v *vm.VM) {
	v.SetGlobal("assert", vm.NewNativeFunc(func(v *vm.VM) int {
		val := v.Get(1)
		if val.IsNil() || (val.IsBool() && !val.AsBool()) {
			msg := v.Get(2)
			if msg.IsString() {
				panic(msg.AsString())
			}
			panic("assertion failed!")
		}
		n := v.ArgCount()
		for i := 0; i < n; i++ {
			v.Set(i, v.Get(i+1))
		}
		return n
	}))
	v.SetGlobal("pcall", vm.NewNativeFunc(func(v *vm.VM) int {
		fn := v.Get(1)
		args := make([]vm.Value, v.ArgCount()-1)
		for i := range args {
			args[i] = v.Get(i + 2)
		}
		results, err := v.ProtectedCall(fn, args)
		if err != nil {
			v.Set(0, vm.False)
			v.Set(1, vm.NewString(err.Error()))
			return 2
		}
		v.Set(0, vm.True)
		for i, r := range results {
			v.Set(i+1, r)
		}
		return len(results) + 1
	}))
	v.SetGlobal("tostring", vm.NewNativeFunc(func(v *vm.VM) int {
		v.Set(0, vm.NewString(v.Get(1).String()))
		return 1
	}))
	v.SetGlobal("type", vm.NewNativeFunc(func(v *vm.VM) int {
		v.Set(0, vm.NewString(v.Get(1).Type()))
		return 1
	}))
}
