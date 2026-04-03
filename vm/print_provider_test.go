package vm

import (
	"context"
	"testing"

	"github.com/iceisfun/golua/v2/compiler"
	"github.com/iceisfun/golua/v2/parser"
)

// capturePrintProvider records all Print and Warn calls for test assertions.
type capturePrintProvider struct {
	prints []string
	warns  []string
}

func (c *capturePrintProvider) Print(_ context.Context, msg string) { c.prints = append(c.prints, msg) }
func (c *capturePrintProvider) Warn(_ context.Context, msg string)  { c.warns = append(c.warns, msg) }

// runLuaWithProvider compiles and runs Lua code with a print provider set,
// plus a minimal print/warn registered as globals (since vm tests can't
// import stdlib due to circular deps).
func runLuaWithProvider(t *testing.T, source string, provider LuaPrintProvider) *VM {
	t.Helper()

	block, err := parser.Parse("test", source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	proto, err := compiler.Compile("test", block)
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	v := New()
	v.SetPrintProvider(provider)

	// Register minimal print() that routes through vm.Print
	v.SetGlobal("print", NewNativeFunc(func(v *VM) int {
		n := v.ArgCount()
		msg := ""
		for i := 1; i <= n; i++ {
			if i > 1 {
				msg += "\t"
			}
			msg += v.Get(i).String()
		}
		v.Print(msg)
		return 0
	}))

	// Register minimal warn() that routes through vm.Warn
	v.SetGlobal("warn", NewNativeFunc(func(v *VM) int {
		n := v.ArgCount()
		if n < 1 {
			panic("bad argument #1 to 'warn'")
		}
		first := v.Get(1).AsString()
		if len(first) > 0 && first[0] == '@' {
			switch first {
			case "@off":
				v.SetWarnEnabled(false)
			case "@on":
				v.SetWarnEnabled(true)
			}
			return 0
		}
		if v.WarnEnabled() {
			msg := "Lua warning: " + first
			for i := 2; i <= n; i++ {
				msg += v.Get(i).AsString()
			}
			v.Warn(msg)
		}
		return 0
	}))

	_, err = v.Run(proto)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}
	return v
}

func TestPrintProviderInterceptsPrint(t *testing.T) {
	cap := &capturePrintProvider{}
	runLuaWithProvider(t, `print("hello")`, cap)

	if len(cap.prints) != 1 {
		t.Fatalf("expected 1 print, got %d", len(cap.prints))
	}
	if cap.prints[0] != "hello" {
		t.Errorf("expected %q, got %q", "hello", cap.prints[0])
	}
}

func TestPrintProviderInterceptsWarn(t *testing.T) {
	cap := &capturePrintProvider{}
	runLuaWithProvider(t, `warn("@on"); warn("danger")`, cap)

	if len(cap.warns) != 1 {
		t.Fatalf("expected 1 warn, got %d", len(cap.warns))
	}
	if cap.warns[0] != "Lua warning: danger" {
		t.Errorf("expected %q, got %q", "Lua warning: danger", cap.warns[0])
	}
}

func TestPrintProviderWarnOff(t *testing.T) {
	cap := &capturePrintProvider{}
	runLuaWithProvider(t, `
		warn("@on")
		warn("first")
		warn("@off")
		warn("silenced")
		warn("@on")
		warn("third")
	`, cap)

	if len(cap.warns) != 2 {
		t.Fatalf("expected 2 warns, got %d: %v", len(cap.warns), cap.warns)
	}
	if cap.warns[0] != "Lua warning: first" {
		t.Errorf("warn[0]: expected %q, got %q", "Lua warning: first", cap.warns[0])
	}
	if cap.warns[1] != "Lua warning: third" {
		t.Errorf("warn[1]: expected %q, got %q", "Lua warning: third", cap.warns[1])
	}
}

func TestDefaultPrintProviderExists(t *testing.T) {
	p := NewDefaultPrintProvider()
	if p == nil {
		t.Fatal("NewDefaultPrintProvider returned nil")
	}
	// Verify it satisfies the interface.
	var _ LuaPrintProvider = p
}

func TestWarnEnabledDefault(t *testing.T) {
	v := New()
	if v.WarnEnabled() {
		t.Error("expected WarnEnabled to be false by default (Lua 5.4 starts with warnings off)")
	}
}

func TestWarnEnabledPerVM(t *testing.T) {
	v1 := New()
	v2 := New()

	v1.SetWarnEnabled(true)
	v2.SetWarnEnabled(true)

	v1.SetWarnEnabled(false)

	if v1.WarnEnabled() {
		t.Error("v1 should have warn disabled")
	}
	if !v2.WarnEnabled() {
		t.Error("v2 should still have warn enabled")
	}
}

func TestPrintProviderInheritedByCoroutine(t *testing.T) {
	cap := &capturePrintProvider{}
	parent := New()
	parent.SetPrintProvider(cap)

	child := NewCoroutineVM(parent, nil, nil, 1)

	if child.PrintProvider() != cap {
		t.Error("coroutine should inherit parent's print provider")
	}
	if child.WarnEnabled() != parent.WarnEnabled() {
		t.Error("coroutine should inherit parent's warn enabled state")
	}
}

func TestPrintProviderNilFallback(t *testing.T) {
	// When no provider is set and capture is enabled, Print still uses capture buffer.
	v := New(WithCaptureOutput(true))
	v.Print("captured line")

	lines := v.OutputLines()
	if len(lines) != 1 || lines[0] != "captured line" {
		t.Errorf("expected captured output, got %v", lines)
	}
}
