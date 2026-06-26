package language

import (
	"testing"

	"github.com/iceisfun/golua/v2/parser"
)

// visibleNames returns the set of symbol names visible at the given position,
// excluding pre-populated stdlib globals (which carry Pos.Line == 0).
func visibleNames(t *testing.T, src string, line, col int) map[string]SymbolKind {
	t.Helper()
	block, _ := parser.ParsePartial("test", src)
	st := Analyze(block, src)
	out := make(map[string]SymbolKind)
	for _, s := range st.VisibleAt(line, col) {
		if s.Pos.Line == 0 {
			continue // stdlib global
		}
		out[s.Name] = s.Kind
	}
	return out
}

func TestScopeFunctionParamsAreLocalToBody(t *testing.T) {
	src := `local function greet(name)
    return name
end

print(greet)
`
	// Inside the body (line 2), the parameter is visible.
	inside := visibleNames(t, src, 2, 12)
	if _, ok := inside["name"]; !ok {
		t.Errorf("parameter 'name' should be visible inside the function body; got %v", inside)
	}
	if k, ok := inside["greet"]; !ok || k != KindFunction {
		t.Errorf("function 'greet' should be visible inside its own body (recursion); got %v", inside)
	}

	// After the function (line 5), the parameter must NOT leak out.
	outside := visibleNames(t, src, 5, 1)
	if _, ok := outside["name"]; ok {
		t.Errorf("parameter 'name' must not be visible after the function ends; got %v", outside)
	}
	if _, ok := outside["greet"]; !ok {
		t.Errorf("function 'greet' should be visible after its definition; got %v", outside)
	}
}

func TestScopeForVarBoundToLoop(t *testing.T) {
	src := `for i = 1, 10 do
    print(i)
end
print(done)
`
	inside := visibleNames(t, src, 2, 12)
	if k, ok := inside["i"]; !ok || k != KindForVar {
		t.Errorf("loop variable 'i' should be visible inside the loop; got %v", inside)
	}
	outside := visibleNames(t, src, 4, 1)
	if _, ok := outside["i"]; ok {
		t.Errorf("loop variable 'i' must not be visible after the loop; got %v", outside)
	}
}

func TestScopeLocalNotVisibleBeforeDeclaration(t *testing.T) {
	src := `local a = 1
local b = 2
`
	// At the start of line 1 (col 1), 'a' is not yet declared.
	early := visibleNames(t, src, 1, 1)
	if _, ok := early["a"]; ok {
		t.Errorf("'a' should not be visible before its declaration; got %v", early)
	}
	// By line 2, 'a' is visible.
	later := visibleNames(t, src, 2, 1)
	if _, ok := later["a"]; !ok {
		t.Errorf("'a' should be visible on line 2; got %v", later)
	}
}

func TestScopeNestedBlockDoesNotLeak(t *testing.T) {
	src := `do
    local secret = 1
    print(secret)
end
print(secret)
`
	inside := visibleNames(t, src, 3, 11)
	if _, ok := inside["secret"]; !ok {
		t.Errorf("'secret' should be visible inside the do-block; got %v", inside)
	}
	outside := visibleNames(t, src, 5, 1)
	if _, ok := outside["secret"]; ok {
		t.Errorf("'secret' must not leak out of the do-block; got %v", outside)
	}
}
