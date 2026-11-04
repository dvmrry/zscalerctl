//go:build race

package cli

// raceEnabled reports whether the race detector is compiled in. The
// catalog-wide render parity walk skips under race: it runs in one goroutine,
// so race finds nothing in it, and instrumentation slowed it past the package
// test timeout. The plain test run covers it in full.
const raceEnabled = true
