package bench

import "testing"

func BenchmarkSort(b *testing.B) {
	benchmarkScript(b, "sort.lua")
}

func BenchmarkTableChurn(b *testing.B) {
	benchmarkScript(b, "table_churn.lua")
}
