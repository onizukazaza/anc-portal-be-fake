package testkit

import (
	"errors"
	"fmt"
	"testing"
)

// mockTB is a minimal testing.TB that captures Fatal/Error calls.
type mockTB struct {
	testing.TB
	failed  bool
	fataled bool
	msg     string
}

func (m *mockTB) Helper()                        {}
func (m *mockTB) Errorf(format string, a ...any) { m.failed = true; m.msg = fmt.Sprintf(format, a...) }
func (m *mockTB) Fatalf(format string, a ...any) { m.fataled = true; m.msg = fmt.Sprintf(format, a...) }

// --- Equal ---

func TestEqual_Pass(t *testing.T) {
	m := &mockTB{}
	Equal(m, 42, 42)
	if m.fataled {
		t.Fatal("Equal should pass for identical values")
	}
}

func TestEqual_Fail(t *testing.T) {
	m := &mockTB{}
	Equal(m, 1, 2)
	if !m.fataled {
		t.Fatal("Equal should fail for different values")
	}
}

func TestEqual_WithMessage(t *testing.T) {
	m := &mockTB{}
	Equal(m, "a", "b", "context %d", 1)
	if !m.fataled {
		t.Fatal("should fail")
	}
	if m.msg == "" {
		t.Fatal("should have message")
	}
}

// --- NotEqual ---

func TestNotEqual_Pass(t *testing.T) {
	m := &mockTB{}
	NotEqual(m, 1, 2)
	if m.fataled {
		t.Fatal("NotEqual should pass for different values")
	}
}

func TestNotEqual_Fail(t *testing.T) {
	m := &mockTB{}
	NotEqual(m, 1, 1)
	if !m.fataled {
		t.Fatal("NotEqual should fail for identical values")
	}
}

// --- True / False ---

func TestTrue_Pass(t *testing.T) {
	m := &mockTB{}
	True(m, true)
	if m.fataled {
		t.Fatal("True should pass")
	}
}

func TestTrue_Fail(t *testing.T) {
	m := &mockTB{}
	True(m, false)
	if !m.fataled {
		t.Fatal("True should fail for false")
	}
}

func TestFalse_Pass(t *testing.T) {
	m := &mockTB{}
	False(m, false)
	if m.fataled {
		t.Fatal("False should pass")
	}
}

func TestFalse_Fail(t *testing.T) {
	m := &mockTB{}
	False(m, true)
	if !m.fataled {
		t.Fatal("False should fail for true")
	}
}

// --- Nil / NotNil ---

func TestNil_Pass(t *testing.T) {
	m := &mockTB{}
	Nil(m, nil)
	if m.fataled {
		t.Fatal("Nil should pass")
	}
}

func TestNil_Fail(t *testing.T) {
	m := &mockTB{}
	Nil(m, "not nil")
	if !m.fataled {
		t.Fatal("Nil should fail for non-nil")
	}
}

func TestNotNil_Pass(t *testing.T) {
	m := &mockTB{}
	NotNil(m, "something")
	if m.fataled {
		t.Fatal("NotNil should pass")
	}
}

func TestNotNil_Fail(t *testing.T) {
	m := &mockTB{}
	NotNil(m, nil)
	if !m.fataled {
		t.Fatal("NotNil should fail for nil")
	}
}

// --- NoError / Error / ErrorIs ---

func TestNoError_Pass(t *testing.T) {
	m := &mockTB{}
	NoError(m, nil)
	if m.fataled {
		t.Fatal("NoError should pass")
	}
}

func TestNoError_Fail(t *testing.T) {
	m := &mockTB{}
	NoError(m, errors.New("boom"))
	if !m.fataled {
		t.Fatal("NoError should fail")
	}
}

func TestError_Pass(t *testing.T) {
	m := &mockTB{}
	Error(m, errors.New("boom"))
	if m.fataled {
		t.Fatal("Error should pass")
	}
}

func TestError_Fail(t *testing.T) {
	m := &mockTB{}
	Error(m, nil)
	if !m.fataled {
		t.Fatal("Error should fail for nil error")
	}
}

func TestErrorIs_Pass(t *testing.T) {
	sentinel := errors.New("sentinel")
	m := &mockTB{}
	ErrorIs(m, fmt.Errorf("wrap: %w", sentinel), sentinel)
	if m.fataled {
		t.Fatal("ErrorIs should pass for wrapped sentinel")
	}
}

func TestErrorIs_Fail(t *testing.T) {
	m := &mockTB{}
	ErrorIs(m, errors.New("a"), errors.New("b"))
	if !m.fataled {
		t.Fatal("ErrorIs should fail for different errors")
	}
}

// --- Contains ---

func TestContains_Pass(t *testing.T) {
	m := &mockTB{}
	Contains(m, "hello world", "world")
	if m.fataled {
		t.Fatal("Contains should pass")
	}
}

func TestContains_Fail(t *testing.T) {
	m := &mockTB{}
	Contains(m, "hello", "world")
	if !m.fataled {
		t.Fatal("Contains should fail")
	}
}

// --- Len ---

func TestLen_Pass(t *testing.T) {
	m := &mockTB{}
	Len(m, []int{1, 2, 3}, 3)
	if m.fataled {
		t.Fatal("Len should pass")
	}
}

func TestLen_Fail(t *testing.T) {
	m := &mockTB{}
	Len(m, []int{1}, 5)
	if !m.fataled {
		t.Fatal("Len should fail")
	}
}

// --- Must variants ---

func TestMustEqual_Pass(t *testing.T) {
	m := &mockTB{}
	MustEqual(m, "ok", "ok")
	if m.fataled {
		t.Fatal("MustEqual should pass")
	}
}

func TestMustEqual_Fail(t *testing.T) {
	m := &mockTB{}
	MustEqual(m, "a", "b")
	if !m.fataled {
		t.Fatal("MustEqual should fatal")
	}
}

func TestMustNoError_Pass(t *testing.T) {
	m := &mockTB{}
	MustNoError(m, nil)
	if m.fataled {
		t.Fatal("MustNoError should pass")
	}
}

func TestMustNoError_Fail(t *testing.T) {
	m := &mockTB{}
	MustNoError(m, errors.New("oops"))
	if !m.fataled {
		t.Fatal("MustNoError should fatal")
	}
}

func TestMustNil_InterfaceWrappedNil(t *testing.T) {
	m := &mockTB{}
	var p *int
	MustNil(m, p)
	if m.fataled {
		t.Fatal("MustNil should pass for (*int)(nil) wrapped in interface")
	}
}

func TestMustNotNil_Pass(t *testing.T) {
	m := &mockTB{}
	v := 42
	MustNotNil(m, &v)
	if m.fataled {
		t.Fatal("MustNotNil should pass for non-nil pointer")
	}
}

// --- formatPrefix ---

func TestFormatPrefix_Empty(t *testing.T) {
	got := formatPrefix(nil)
	if got != "" {
		t.Fatalf("want empty, got %q", got)
	}
}

func TestFormatPrefix_Single(t *testing.T) {
	got := formatPrefix([]any{"label"})
	if got != "label: " {
		t.Fatalf("want %q, got %q", "label: ", got)
	}
}

func TestFormatPrefix_Formatted(t *testing.T) {
	got := formatPrefix([]any{"step %d", 3})
	if got != "step 3: " {
		t.Fatalf("want %q, got %q", "step 3: ", got)
	}
}
