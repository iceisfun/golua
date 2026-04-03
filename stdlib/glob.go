package stdlib

import (
	"fmt"

	"github.com/iceisfun/golua/v2/glob"
	"github.com/iceisfun/golua/v2/vm"
)

func openGlob(v *vm.VM) {
	g := vm.NewEmptyTable()

	g.SetString("match", vm.NewNativeFunc(globMatch))
	g.SetString("match_words", vm.NewNativeFunc(globMatchWords))
	g.SetString("match_named", vm.NewNativeFunc(globMatchNamed))
	g.SetString("has_pattern", vm.NewNativeFunc(globHasPattern))

	v.SetGlobal("glob", vm.NewTable(g))
}

// glob.match(pattern, name) -> boolean
func globMatch(v *vm.VM) int {
	pattern := getString(v, 1, "glob.match")
	name := getString(v, 2, "glob.match")
	matched, err := glob.Match(pattern, name)
	if err != nil {
		callerArgError(v, 1, "glob.match", fmt.Sprintf("%s", err))
	}
	v.Set(0, vm.NewBool(matched))
	return 1
}

// glob.match_words(pattern, name) -> boolean
func globMatchWords(v *vm.VM) int {
	pattern := getString(v, 1, "glob.match_words")
	name := getString(v, 2, "glob.match_words")
	matched, err := glob.MatchWords(pattern, name)
	if err != nil {
		callerArgError(v, 1, "glob.match_words", fmt.Sprintf("%s", err))
	}
	v.Set(0, vm.NewBool(matched))
	return 1
}

// glob.match_named(pattern, text) -> boolean, table
func globMatchNamed(v *vm.VM) int {
	pattern := getString(v, 1, "glob.match_named")
	text := getString(v, 2, "glob.match_named")
	ok, caps, err := glob.MatchNamed(pattern, text)
	if err != nil {
		callerArgError(v, 1, "glob.match_named", fmt.Sprintf("%s", err))
	}
	v.Set(0, vm.NewBool(ok))
	t := vm.NewEmptyTable()
	for k, val := range caps {
		t.SetString(k, vm.NewString(val))
	}
	v.Set(1, vm.NewTable(t))
	return 2
}

// glob.has_pattern(s) -> boolean
func globHasPattern(v *vm.VM) int {
	s := getString(v, 1, "glob.has_pattern")
	v.Set(0, vm.NewBool(glob.HasPatternCharacters(s)))
	return 1
}
