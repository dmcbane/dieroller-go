// Package version is the single source of truth for the version both commands
// report. All four implementations of dieroller report the same version.
package version

// Version is overridden at build time with
// -ldflags "-X github.com/dmcbane/dieroller-go/internal/version.Version=x.y.z".
var Version = "1.1.0"
