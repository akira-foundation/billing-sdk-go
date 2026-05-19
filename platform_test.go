package billing

import (
	"strings"
	"testing"
)

func TestPlatformSlug(t *testing.T) {
	cases := []struct {
		name string
		p    Platform
		want string
	}{
		{"mac arm", Platform{OS: OSMacOS, Arch: ArchArm64}, "macos-arm64"},
		{"linux x86", Platform{OS: OSLinux, Arch: ArchX86_64}, "linux-x86_64"},
		{"windows x86", Platform{OS: OSWindows, Arch: ArchX86_64}, "windows-x86_64"},
	}
	for _, tc := range cases {
		if got := tc.p.Slug(); got != tc.want {
			t.Fatalf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestPlatformPredicates(t *testing.T) {
	mac := Platform{OS: OSMacOS, Arch: ArchArm64}
	lin := Platform{OS: OSLinux, Arch: ArchX86_64}
	win := Platform{OS: OSWindows, Arch: ArchX86_64}

	if !mac.IsMacOS() || mac.IsLinux() || mac.IsWindows() {
		t.Fatal("macOS predicate mismatch")
	}
	if !lin.IsLinux() || lin.IsMacOS() {
		t.Fatal("Linux predicate mismatch")
	}
	if !win.IsWindows() || win.IsLinux() {
		t.Fatal("Windows predicate mismatch")
	}
}

func TestOSFromTarget(t *testing.T) {
	cases := map[string]OS{
		"darwin":  OSMacOS,
		"macos":   OSMacOS,
		"linux":   OSLinux,
		"windows": OSWindows,
		"win32":   OSWindows,
	}
	for input, want := range cases {
		got, ok := OSFromTarget(input)
		if !ok || got != want {
			t.Fatalf("OSFromTarget(%q) = (%q, %v); want (%q, true)", input, got, ok, want)
		}
	}
	if _, ok := OSFromTarget("haiku"); ok {
		t.Fatal("OSFromTarget should reject unsupported values")
	}
}

func TestArchFromTarget(t *testing.T) {
	cases := map[string]Arch{
		"arm64":   ArchArm64,
		"aarch64": ArchArm64,
		"amd64":   ArchX86_64,
		"x86_64":  ArchX86_64,
		"x64":     ArchX86_64,
	}
	for input, want := range cases {
		got, ok := ArchFromTarget(input)
		if !ok || got != want {
			t.Fatalf("ArchFromTarget(%q) = (%q, %v); want (%q, true)", input, got, ok, want)
		}
	}
	if _, ok := ArchFromTarget("riscv"); ok {
		t.Fatal("ArchFromTarget should reject unsupported values")
	}
}

func TestDetectPlatform(t *testing.T) {
	if _, ok := DetectPlatform(); !ok {
		t.Fatal("DetectPlatform should match the host target on supported runners")
	}
}

func TestDownloadURL(t *testing.T) {
	p := Platform{OS: OSMacOS, Arch: ArchArm64}
	want := "https://billing.test/api/v1/downloads/unified-dev/stable/macos-arm64"

	if got := DownloadURL("https://billing.test", "unified-dev", "stable", p); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if got := DownloadURL("https://billing.test/", "unified-dev", "stable", p); got != want {
		t.Fatalf("trailing slash not trimmed: got %q", got)
	}
}

func TestPickDownloadURL(t *testing.T) {
	url, ok := PickDownloadURL("https://billing.test", "unified-dev", "stable", nil)
	if !ok {
		t.Fatal("PickDownloadURL should succeed when host target is supported")
	}
	if !strings.HasPrefix(url, "https://billing.test/api/v1/downloads/unified-dev/stable/") {
		t.Fatalf("unexpected url shape: %q", url)
	}
}
