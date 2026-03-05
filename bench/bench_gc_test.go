package bench

import "testing"

func BenchmarkStringConcat(b *testing.B) {
	benchmarkScript(b, "string_concat.lua")
}
