package language

import (
	"sort"
	"strings"
	"testing"

	"github.com/iceisfun/golua/v2/stdlib"
	"github.com/iceisfun/golua/v2/vm"
)

// TestStdlibMetadataMatchesVM keeps the hand-written completion/hover metadata
// in stdlib.go honest against the real library. It opens a VM configured
// exactly like the Run sandbox in main.go (stdlib.Open with no providers) and
// fails if the curated tables and the live VM disagree about which symbols
// exist. This is the drift guard referenced in stdlib.go: when someone adds or
// removes a stdlib function, this test points at the metadata that needs the
// matching edit.
//
// It deliberately checks the *set* of names, not signatures or docs (those are
// human prose the VM cannot supply).
func TestStdlibMetadataMatchesVM(t *testing.T) {
	v := vm.New()
	stdlib.Open(v) // same modules the Run sandbox registers (no providers)

	real := tableNames(t, v.Globals())

	// Internal output-capture helpers are host plumbing, not Lua API surface,
	// so they are intentionally excluded from completion.
	internal := map[string]bool{"_lastoutput": true, "_outputlines": true}

	curatedGlobals := make(map[string]bool, len(globals))
	for _, e := range globals {
		curatedGlobals[e.Name] = true
	}

	// Every global the VM exposes (minus internals) must be documented.
	for _, name := range real {
		if internal[name] || curatedGlobals[name] {
			continue
		}
		t.Errorf("global %q exists in the VM but is missing from stdlib.go metadata", name)
	}
	// Every documented global must really exist.
	for name := range curatedGlobals {
		if v.GetGlobal(name).IsNil() {
			t.Errorf("global %q is in stdlib.go metadata but does not exist in the VM", name)
		}
	}

	// Check each curated library table member-for-member against the VM.
	for parent, members := range tables {
		g := v.GetGlobal(parent)
		if !g.IsTable() {
			t.Errorf("library table %q is documented but not a table in the VM", parent)
			continue
		}
		realMembers := tableNames(t, g.AsTable())
		realSet := make(map[string]bool, len(realMembers))
		for _, n := range realMembers {
			realSet[n] = true
		}
		curatedSet := make(map[string]bool, len(members))
		for _, e := range members {
			curatedSet[e.Name] = true
			if !realSet[e.Name] {
				t.Errorf("%s.%s is documented but missing from the VM", parent, e.Name)
			}
		}
		for _, n := range realMembers {
			if !curatedSet[n] {
				t.Errorf("%s.%s exists in the VM but is missing from stdlib.go metadata", parent, n)
			}
		}
	}
}

// tableNames returns the string keys of a Lua table, sorted for stable output.
func tableNames(t *testing.T, tbl vm.LuaTable) []string {
	t.Helper()
	var names []string
	key := vm.Nil
	for {
		next, _, err := tbl.Next(key)
		if err != nil {
			t.Fatalf("table iteration failed: %v", err)
		}
		if next.IsNil() {
			break
		}
		if next.IsString() {
			names = append(names, next.AsString())
		}
		key = next
	}
	sort.Strings(names)
	return names
}

// TestStdlibLookupConsistency verifies the fully-qualified lookup index stays in
// sync with the globals/tables slices it is built from.
func TestStdlibLookupConsistency(t *testing.T) {
	for _, e := range globals {
		if _, ok := StdlibLookup(e.Name); !ok {
			t.Errorf("global %q not found via StdlibLookup", e.Name)
		}
	}
	for parent, members := range tables {
		for _, e := range members {
			fq := parent + "." + e.Name
			got, ok := StdlibLookup(fq)
			if !ok {
				t.Errorf("%q not found via StdlibLookup", fq)
				continue
			}
			if got.Parent != parent {
				t.Errorf("%q has Parent %q, want %q", fq, got.Parent, parent)
			}
			if !strings.HasPrefix(got.Signature, parent+".") && got.Kind != "value" {
				// Signatures for members are conventionally "parent.name(...)".
				t.Logf("note: %q signature %q does not start with %q.", fq, got.Signature, parent)
			}
		}
	}
}
