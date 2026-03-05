package bench

import "testing"

func BenchmarkPrimes(b *testing.B) {
	benchmarkScript(b, "primes.lua")
}

func BenchmarkNBody(b *testing.B) {
	benchmarkScript(b, "nbody.lua")
}

func BenchmarkClosureAlloc(b *testing.B) {
	benchmarkScript(b, "closure_alloc.lua")
}

func BenchmarkMetamethod(b *testing.B) {
	benchmarkScript(b, "metamethod.lua")
}
