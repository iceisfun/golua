package golua_test

import (
	"fmt"
	"runtime/debug"
	"strings"
	"testing"
)

// Machine-generated Lua — config dumps, transpiler output, serialized data —
// routinely contains constructs no hand-written file would: an operator chain
// with a million terms, or a file that is almost entirely comments. The lexer
// and the compiler must walk those iteratively; consuming a Go frame per
// comment or per chain link ends the process with a fatal stack overflow that
// no pcall can catch. The tests below cap the goroutine stack so a regression
// trips on a short chunk instead of needing a gigabyte of stack to show up.

func TestLargeGeneratedChunkRuns(t *testing.T) {
	defer debug.SetMaxStack(debug.SetMaxStack(32 << 20))

	const terms = 60000
	const comments = 200000
	src := "local a = 1\n" +
		"local total = a" + strings.Repeat("+a", terms) + "\n" +
		strings.Repeat("--\n", comments) +
		fmt.Sprintf("assert(total == %d, 'wrong total: ' .. tostring(total))\n", terms+1)

	runLuaSource(t, src, "generated")
}

func TestLoadOfLargeGeneratedChunk(t *testing.T) {
	defer debug.SetMaxStack(debug.SetMaxStack(32 << 20))

	runLuaSource(t, `
		local chain = assert(load("local a = 1 return a" .. string.rep("+a", 60000), "=chain"))
		assert(chain() == 60001)

		local comments = assert(load(string.rep("--\n", 200000) .. "return 42", "=comments"))
		assert(comments() == 42)

		local fields = assert(load("local t = {} return t" .. string.rep(".f", 60000), "=fields"))
		local ok, err = pcall(fields)
		assert(not ok and err:find("attempt to index a nil value"), tostring(err))
	`, "loader")
}

// A chain is flat, but generated code also nests: parentheses, table
// constructors, functions and blocks inside one another. Reference Lua bounds
// that nesting and reports "C stack overflow", a value load() hands back like
// any other syntax error. Nesting must therefore neither compile silently nor
// end the process, however deep it goes.
func TestDeeplyNestedChunkIsACatchableError(t *testing.T) {
	defer debug.SetMaxStack(debug.SetMaxStack(32 << 20))

	runLuaSource(t, `
		local n = 5000
		local shapes = {
			{"parens", "return " .. string.rep("(", n) .. "1" .. string.rep(")", n)},
			{"tables", "return " .. string.rep("{", n) .. string.rep("}", n)},
			{"not", "return " .. string.rep("not ", n) .. "1"},
			{"functions", string.rep("return function() ", n) .. "return 1 " .. string.rep("end ", n)},
			{"blocks", string.rep("if true then ", n) .. "return 1 " .. string.rep("end ", n)},
			{"concat", "local a = 'x' return a" .. string.rep(" .. a", n)},
		}
		for _, shape in ipairs(shapes) do
			local name, src = shape[1], shape[2]
			local f, err = load(src, "=" .. name)
			assert(f == nil, name .. ": expected the nesting limit to reject this")
			assert(err:find("C stack overflow"), name .. ": " .. tostring(err))
		end
	`, "nested")
}

// A suffixed chain nests down its left spine, so every node's End() used to
// walk the whole chain when asked for its own end position — even though the
// parser had already recorded one. Asking once, from the table constructor that
// holds the chain, was enough to exhaust the goroutine stack.
func TestEndPositionOfLongChainIsNotWalked(t *testing.T) {
	defer debug.SetMaxStack(debug.SetMaxStack(32 << 20))

	runLuaSource(t, `
		local src = "local t = {} local r = { x = t" .. string.rep(".f", 200000) .. " }"
		local f, err = load(src, "=chain")
		assert(f ~= nil, tostring(err))
	`, "chain_end")
}
