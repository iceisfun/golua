// Example: Directive-driven script loader.
//
// A common embedder pattern: a directory contains many Lua scripts,
// each with a header describing how the host should treat it
// (scheduler interval, scope name, enable/disable flag). The host
// reads the header before deciding whether to compile and run.
//
// Directives are a golua-specific extension; the reference Lua
// interpreter ignores them as ordinary comments. See the README.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/iceisfun/golua/v2/compiler"
	"github.com/iceisfun/golua/v2/directives"
	"github.com/iceisfun/golua/v2/parser"
	"github.com/iceisfun/golua/v2/stdlib"
	"github.com/iceisfun/golua/v2/vm"
)

type script struct {
	path  string
	name  string
	scope string
	tick  time.Duration
	src   string
}

// loadScript reads a Lua file, parses its directive header, and
// applies embedder policy. Returns ok=false if the script is disabled.
func loadScript(path string) (*script, bool, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, false, err
	}
	src := string(raw)

	f, err := directives.Parse(src)
	if err != nil {
		return nil, false, err
	}

	// Policy lives here in the embedder, not in the directives package.
	if f.Has("disabled") {
		return nil, false, nil
	}

	s := &script{
		path: path,
		name: filepath.Base(path),
		src:  src,
	}
	if v, ok := f.Get("scope"); ok {
		s.scope = v
	}
	if v, ok := f.Get("tick"); ok {
		d, err := time.ParseDuration(v)
		if err != nil {
			return nil, false, fmt.Errorf("%s: bad @tick %q: %w", path, v, err)
		}
		s.tick = d
	}
	return s, true, nil
}

func runOnce(s *script) error {
	block, err := parser.Parse(s.name, s.src)
	if err != nil {
		return err
	}
	proto, err := compiler.Compile(s.name, block)
	if err != nil {
		return err
	}
	v := vm.New()
	stdlib.Open(v)
	_, err = v.Run(proto)
	return err
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("usage: directive_loader <dir>")
	}
	matches, err := filepath.Glob(filepath.Join(os.Args[1], "*.lua"))
	if err != nil {
		log.Fatal(err)
	}
	sort.Strings(matches)

	for _, p := range matches {
		s, enabled, err := loadScript(p)
		if err != nil {
			log.Printf("skip %s: %v", p, err)
			continue
		}
		if !enabled {
			fmt.Printf("disabled: %s\n", filepath.Base(p))
			continue
		}
		fmt.Printf("loaded:   %-12s scope=%-12s tick=%s\n", s.name, s.scope, s.tick)
		if err := runOnce(s); err != nil {
			log.Printf("run %s: %v", p, err)
		}
	}
}
