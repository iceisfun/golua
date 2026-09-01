package vm

import (
	"fmt"
	"testing"
)

// buildStrTable fills a table with n string keys and returns it.
func buildStrTable(n int) *Table {
	t := NewEmptyTable()
	for i := 0; i < n; i++ {
		t.SetString(fmt.Sprintf("key_%05d", i), NewInt(int64(i)))
	}
	return t
}

// benchmarkTraverse walks a table of n string keys from end to end, which is
// the path pairs() takes. It is the cost a table pays for being iterated.
func benchmarkTraverse(b *testing.B, n int) {
	t := buildStrTable(n)
	b.ReportAllocs()
	b.ResetTimer()
	visited := 0
	for i := 0; i < b.N; i++ {
		k, _, _ := t.Next(Nil)
		for !k.IsNil() {
			visited++
			k, _, _ = t.Next(k)
		}
	}
	if visited != b.N*n {
		b.Fatalf("visited %d entries, want %d", visited, b.N*n)
	}
}

func BenchmarkTableTraverse64(b *testing.B)   { benchmarkTraverse(b, 64) }
func BenchmarkTableTraverse1024(b *testing.B) { benchmarkTraverse(b, 1024) }
func BenchmarkTableTraverse8192(b *testing.B) { benchmarkTraverse(b, 8192) }

// BenchmarkTableTraverseInterleaved walks one table with two traversals at a
// time, stepping them alternately. One cursor cannot serve both, so this is
// the case that resolves a key to its slot the hard way — and the one that
// decides whether doing so stays a constant-cost step.
func BenchmarkTableTraverseInterleaved(b *testing.B) {
	const n = 1024
	t := buildStrTable(n)
	b.ReportAllocs()
	b.ResetTimer()
	visited := 0
	for i := 0; i < b.N; i++ {
		a, _, _ := t.Next(Nil)
		c, _, _ := t.Next(Nil)
		for !a.IsNil() || !c.IsNil() {
			if !a.IsNil() {
				visited++
				a, _, _ = t.Next(a)
			}
			if !c.IsNil() {
				visited++
				c, _, _ = t.Next(c)
			}
		}
	}
	if visited != b.N*n*2 {
		b.Fatalf("visited %d entries, want %d", visited, b.N*n*2)
	}
}

// benchmarkBuild fills a fresh table with n string keys and never walks it. A
// table nobody iterates must not be charged for the machinery that makes
// iteration cheap, so this is the benchmark that has to stay flat.
func benchmarkBuild(b *testing.B, n int) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buildStrTable(n)
	}
}

func BenchmarkTableBuild32(b *testing.B)   { benchmarkBuild(b, 32) }
func BenchmarkTableBuild1000(b *testing.B) { benchmarkBuild(b, 1000) }
