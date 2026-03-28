// Package testkit provides generic test assertion helpers for reducing
// boilerplate in unit tests without external dependencies.
//
// Before:
//
//	if got != want {
//	    t.Fatalf("want %v, got %v", want, got)
//	}
//
// After:
//
//	assert.Equal(t, got, want)
package testkit
