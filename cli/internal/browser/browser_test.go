package browser_test

import (
	"errors"
	"testing"

	"github.com/popatkaran/postulate/cli/internal/browser"
)

func TestOpen_InvokesFnWithURL(t *testing.T) {
	var got []string
	fn := func(name string, args ...string) error {
		got = append([]string{name}, args...)
		return nil
	}
	if err := browser.Open("https://example.com", fn); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) == 0 {
		t.Error("expected exec fn to be called")
	}
	// The URL must appear somewhere in the args.
	found := false
	for _, a := range got {
		if a == "https://example.com" {
			found = true
		}
	}
	if !found {
		t.Errorf("URL not passed to exec fn: %v", got)
	}
}

func TestOpen_FnError_ReturnsError(t *testing.T) {
	fn := func(_ string, _ ...string) error { return errors.New("no browser") }
	if err := browser.Open("https://example.com", fn); err == nil {
		t.Error("expected error when fn fails")
	}
}

func TestOpenOrPrint_FnError_PrintsFallback(t *testing.T) {
	fn := func(_ string, _ ...string) error { return errors.New("no browser") }
	var printed string
	browser.OpenOrPrint("https://example.com", fn, func(format string, a ...any) {
		printed += browser.Sprintf(format, a...)
	})
	if printed == "" {
		t.Error("expected fallback message to be printed")
	}
}

func TestOpenOrPrint_FnSuccess_NoPrint(t *testing.T) {
	fn := func(_ string, _ ...string) error { return nil }
	var printed string
	browser.OpenOrPrint("https://example.com", fn, func(format string, a ...any) {
		printed += browser.Sprintf(format, a...)
	})
	if printed != "" {
		t.Errorf("expected no fallback message, got: %s", printed)
	}
}
