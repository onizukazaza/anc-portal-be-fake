package testkit

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// Equal checks that got == want using comparable constraint.
// Fails the test with a descriptive message if they differ.
func Equal[T comparable](t testing.TB, got, want T, msgAndArgs ...any) {
	t.Helper()
	if got != want {
		t.Fatalf("%swant %v, got %v", formatPrefix(msgAndArgs), want, got)
	}
}

// NotEqual checks that got != want.
func NotEqual[T comparable](t testing.TB, got, notWant T, msgAndArgs ...any) {
	t.Helper()
	if got == notWant {
		t.Fatalf("%svalues should differ, both are %v", formatPrefix(msgAndArgs), got)
	}
}

// True checks that value is true.
func True(t testing.TB, value bool, msgAndArgs ...any) {
	t.Helper()
	if !value {
		t.Fatalf("%sexpected true, got false", formatPrefix(msgAndArgs))
	}
}

// False checks that value is false.
func False(t testing.TB, value bool, msgAndArgs ...any) {
	t.Helper()
	if value {
		t.Fatalf("%sexpected false, got true", formatPrefix(msgAndArgs))
	}
}

// Nil checks that value is nil (handles interface-wrapped nil pointers).
func Nil(t testing.TB, value any, msgAndArgs ...any) {
	t.Helper()
	if !isNil(value) {
		t.Fatalf("%sexpected nil, got %v", formatPrefix(msgAndArgs), value)
	}
}

// NotNil checks that value is not nil (handles interface-wrapped nil pointers).
func NotNil(t testing.TB, value any, msgAndArgs ...any) {
	t.Helper()
	if isNil(value) {
		t.Fatalf("%sexpected non-nil value", formatPrefix(msgAndArgs))
	}
}

// isNil checks whether v is nil, handling interface-wrapped nil pointers.
func isNil(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func, reflect.Interface:
		return rv.IsNil()
	}
	return false
}

// NoError checks that err is nil.
func NoError(t testing.TB, err error, msgAndArgs ...any) {
	t.Helper()
	if err != nil {
		t.Fatalf("%sunexpected error: %v", formatPrefix(msgAndArgs), err)
	}
}

// Error checks that err is not nil.
func Error(t testing.TB, err error, msgAndArgs ...any) {
	t.Helper()
	if err == nil {
		t.Fatalf("%sexpected an error, got nil", formatPrefix(msgAndArgs))
	}
}

// ErrorIs checks that errors.Is(err, target) is true.
func ErrorIs(t testing.TB, err, target error, msgAndArgs ...any) {
	t.Helper()
	if !errors.Is(err, target) {
		t.Fatalf("%serror mismatch: want %v, got %v", formatPrefix(msgAndArgs), target, err)
	}
}

// Contains checks that s contains substr.
func Contains(t testing.TB, s, substr string, msgAndArgs ...any) {
	t.Helper()
	if !strings.Contains(s, substr) {
		t.Fatalf("%sstring %q does not contain %q", formatPrefix(msgAndArgs), s, substr)
	}
}

// Len checks that a slice has expected length.
func Len[T any](t testing.TB, slice []T, want int, msgAndArgs ...any) {
	t.Helper()
	if len(slice) != want {
		t.Fatalf("%slen: want %d, got %d", formatPrefix(msgAndArgs), want, len(slice))
	}
}

func formatPrefix(msgAndArgs []any) string {
	if len(msgAndArgs) == 0 {
		return ""
	}
	if len(msgAndArgs) == 1 {
		return fmt.Sprintf("%v: ", msgAndArgs[0])
	}
	return fmt.Sprintf(fmt.Sprintf("%v", msgAndArgs[0]), msgAndArgs[1:]...) + ": "
}
