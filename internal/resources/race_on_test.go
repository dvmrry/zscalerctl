//go:build race

package resources_test

// raceEnabled reports whether the race detector is compiled in. The catalog
// secret-shape sweep skips under race: its 116,000 cases took most of the
// package's race run while adding no concurrency the catalog canary sweep
// does not already exercise. The plain test run covers it in full.
const raceEnabled = true
