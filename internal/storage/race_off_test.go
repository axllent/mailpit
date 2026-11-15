//go:build !race

package storage

// raceEnabled is whether the tests are built with the race detector, which makes some of them very slow.
const raceEnabled = false
