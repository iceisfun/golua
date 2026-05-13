package tests

import (
	"regexp"
	"strings"
	"testing"

	"github.com/iceisfun/golua/stdlib"
	"github.com/iceisfun/golua/vm"
)

func runGoLuaWithDebug(t *testing.T, code string) (string, error) {
	t.Helper()
	proto, err := compileLua("=test", code)
	if err != nil {
		return "", err
	}
	v := vm.New(vm.WithCaptureOutput(true))
	v.SetDebugProvider(vm.NewDefaultDebugProvider())
	stdlib.Open(v)
	_, runErr := v.Run(proto)
	return strings.Join(v.OutputLines(), "\n"), runErr
}

// TestTracebackCoroutineClose verifies that debug.traceback inside a __close
// handler run by coroutine.close does not leak the suspended coroutine frames.
func TestTracebackCoroutineClose(t *testing.T) {
	code := `
local co = coroutine.create(function()
  local obj = setmetatable({}, {
    __close = function()
      print(debug.traceback("TB", 0))
    end,
  })
  local x <close> = obj
  coroutine.yield("pause")
end)
coroutine.resume(co)
coroutine.close(co)
`
	out, err := runGoLuaWithDebug(t, code)
	if err != nil {
		t.Fatalf("GoLua error: %v", err)
	}
	re := regexp.MustCompile("(?ms)^TB\\nstack traceback:\\n\\t\\[C\\]: in function 'debug\\.traceback'\\n\\ttest:\\d+: in function <test:\\d+>\\n?$")
	if !re.MatchString(out) {
		t.Fatalf("unexpected traceback output: %q", out)
	}
	if matched, _ := regexp.MatchString(`coroutine\.yield|metamethod 'close'`, out); matched {
		t.Fatalf("unexpected leaked coroutine/close frame in output: %q", out)
	}
}

// TestXpcallCloseReplacementTraceback verifies that replacement __close errors
// point at the close callback itself instead of the synthetic metamethod label.
func TestXpcallCloseReplacementTraceback(t *testing.T) {
	code := `
local _, msg = xpcall(function()
  local x <close> = setmetatable({}, {
    __close = function()
      error("CLOSE")
    end,
  })
  error("MAIN")
end, debug.traceback)
print(msg)
`
	out, err := runGoLuaWithDebug(t, code)
	if err != nil {
		t.Fatalf("GoLua error: %v", err)
	}
	re := regexp.MustCompile("(?ms)^.*CLOSE\\nstack traceback:\\n\\t\\[C\\]: in function 'error'\\n\\ttest:\\d+: in function <test:\\d+>.*$")
	if !re.MatchString(out) {
		t.Fatalf("unexpected traceback output: %q", out)
	}
	if matched, _ := regexp.MatchString(`metamethod 'close'`, out); matched {
		t.Fatalf("unexpected metamethod label in output: %q", out)
	}
}
