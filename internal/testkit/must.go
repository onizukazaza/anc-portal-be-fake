package testkit

import (
	"errors"
	"testing"
)

// Must functions mirror the Assert functions but call t.FailNow (Fatal)
// instead of t.Fail (Error). Use them for setup preconditions that must
// hold for the rest of the test to make sense.

// MustEqual fatals when got != want.
func MustEqual[T comparable](t testing.TB, got, want T, msgAndArgs ...any) {
	t.Helper()
	if got != want {
		t.Fatalf("%swant %v, got %v", formatPrefix(msgAndArgs), want, got)
	}
}

// MustNoError fatals when err != nil.
func MustNoError(t testing.TB, err error, msgAndArgs ...any) {
	t.Helper()
	if err != nil {
		t.Fatalf("%sunexpected error: %v", formatPrefix(msgAndArgs), err)
	}
}

// MustNil fatals when v != nil.
func MustNil(t testing.TB, v any, msgAndArgs ...any) {
	t.Helper()
	if !isNil(v) {
		t.Fatalf("%sexpected nil, got %v", formatPrefix(msgAndArgs), v)
	}
}

// MustNotNil fatals when v == nil.
func MustNotNil(t testing.TB, v any, msgAndArgs ...any) {
	t.Helper()
	if isNil(v) {
		t.Fatalf("%sexpected non-nil", formatPrefix(msgAndArgs))
	}
}

// MustTrue fatals when v is false.
func MustTrue(t testing.TB, v bool, msgAndArgs ...any) {
	t.Helper()
	if !v {
		t.Fatalf("%sexpected true", formatPrefix(msgAndArgs))
	}
}

// MustErrorIs fatals when !errors.Is(got, want).
func MustErrorIs(t testing.TB, got, want error, msgAndArgs ...any) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Fatalf("%swant error %v, got %v", formatPrefix(msgAndArgs), want, got)
	}
}
