package clickhouseclient

import "testing"

func TestRedactSensitiveValues(t *testing.T) {
	got := redactSensitiveValues(
		"credential=top-secret and token=short",
		[]string{"top-secret", "short", ""},
	)
	want := "credential=[REDACTED] and token=[REDACTED]"
	if got != want {
		t.Fatalf("redactSensitiveValues() = %q, want %q", got, want)
	}

	got = redactSensitiveValues("long=abcdef short=abc", []string{"abc", "abcdef"})
	want = "long=[REDACTED] short=[REDACTED]"
	if got != want {
		t.Fatalf("redactSensitiveValues() with overlapping secrets = %q, want %q", got, want)
	}
}
