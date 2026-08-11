//go:build race

package tests

// raceEnabled reports whether the test binary was built with -race, which
// slows execution enough to matter for luaTestTimeout.
const raceEnabled = true
