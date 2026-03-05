package bench

import "testing"

func BenchmarkCoroutinePingPong(b *testing.B) {
	benchmarkScript(b, "coroutine_pingpong.lua")
}

func BenchmarkBinaryTrees(b *testing.B) {
	benchmarkScript(b, "binarytrees.lua")
}
