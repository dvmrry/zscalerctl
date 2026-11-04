//go:build race

package redact_test

// raceEnabled reports whether the race detector is compiled in. The scan
// linearity checks skip under race: they compare timings, which race
// instrumentation distorts (and slows ~10x, enough to exceed the package test
// timeout). The catch-rate measurements skip too: they run in one goroutine,
// so race finds nothing in them, and they dominated the package's race run.
const raceEnabled = true
