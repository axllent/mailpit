//go:build race

package apiv1

// raceEnabled is whether the tests are built with the race detector, which makes them much slower and use much more
// memory, so the tests which measure either do not run.
const raceEnabled = true
