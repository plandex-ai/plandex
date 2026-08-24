package fs

import (
	"path/filepath"
	"testing"
)

func TestGetHomePlandexDirUsesXDGConfigHome(t *testing.T) {
	got := getHomePlandexDir("/home/me", "/tmp/xdg", "")
	want := filepath.Join("/tmp/xdg", "plandex")

	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestGetHomePlandexDirUsesXDGConfigHomeForDevelopment(t *testing.T) {
	got := getHomePlandexDir("/home/me", "/tmp/xdg", "development")
	want := filepath.Join("/tmp/xdg", "plandex-dev")

	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestGetHomePlandexDirFallsBackToLegacyHomeDir(t *testing.T) {
	got := getHomePlandexDir("/home/me", "", "")
	want := filepath.Join("/home/me", ".plandex-home-v2")

	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestGetHomePlandexDirIgnoresRelativeXDGConfigHome(t *testing.T) {
	got := getHomePlandexDir("/home/me", "relative", "")
	want := filepath.Join("/home/me", ".plandex-home-v2")

	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
