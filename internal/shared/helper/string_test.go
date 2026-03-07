package helper

import (
	"testing"
)

func TestNormalizeSlug(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"spaces to underscores", "hello world", "hello_world"},
		{"multiple spaces", "a  b   c", "a__b___c"},
		{"letters and digits only", "abc123", "abc123"},
		{"removes special chars", "hello-world!go@test", "helloworldgotest"},
		{"keeps underscores", "already_underscore", "already_underscore"},
		{"mixed", "My Project 2024 (v1)", "My_Project_2024_v1"},
		{"unicode letters", "café", "café"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeSlug(tt.in)
			if got != tt.want {
				t.Errorf("NormalizeSlug(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestRandomString(t *testing.T) {
	// UUID string is 36 chars; we slice to n
	t.Run("length", func(t *testing.T) {
		n := 8
		got := RandomString(n)
		if len(got) != n {
			t.Errorf("RandomString(%d) len = %d, want %d", n, len(got), n)
		}
	})
	t.Run("non-empty", func(t *testing.T) {
		got := RandomString(5)
		if got == "" {
			t.Error("RandomString(5) returned empty string")
		}
	})
	t.Run("different each call", func(t *testing.T) {
		a := RandomString(36)
		b := RandomString(36)
		if a == b {
			t.Error("RandomString(36) returned same value twice (unlikely for UUID)")
		}
	})
}
