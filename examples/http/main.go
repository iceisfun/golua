// Example runner for the optional http module.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/iceisfun/golua/v2/compiler"
	"github.com/iceisfun/golua/v2/parser"
	"github.com/iceisfun/golua/v2/stdlib"
	gohttp "github.com/iceisfun/golua/v2/stdlib/http"
	"github.com/iceisfun/golua/v2/vm"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("usage: go run ./examples/http ./examples/http/simple_get.lua")
	}

	path := os.Args[1]
	source, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("read %s: %v", path, err)
	}

	name := filepath.Base(path)
	block, err := parser.Parse(name, string(source))
	if err != nil {
		log.Fatalf("parse %s: %v", path, err)
	}

	proto, err := compiler.Compile(name, block)
	if err != nil {
		log.Fatalf("compile %s: %v", path, err)
	}

	v := vm.New()
	stdlib.Open(v)
	gohttp.Open(v)

	if _, err := v.Run(proto); err != nil {
		log.Fatalf("run %s: %v", path, err)
	}

	fmt.Printf("\ncompleted %s\n", name)
}
