// Package platform provides cross-environment platform detection. The returned
// slug matches the billing AssetPlatform contract used by the downloads
// endpoint and mirrors the JS and Rust SDK shapes.
package platform

import (
	"runtime"
	"strings"
)

// OS is a normalized operating-system identifier matching the billing
// AssetPlatform slug (macos, linux, windows).
type OS string

// Arch is a normalized CPU architecture identifier matching the billing
// AssetPlatform slug (arm64, x86_64).
type Arch string

const (
	OSMacOS   OS = "macos"
	OSLinux   OS = "linux"
	OSWindows OS = "windows"

	ArchArm64  Arch = "arm64"
	ArchX86_64 Arch = "x86_64"
)

// Platform is the canonical (OS, Arch) pair the billing API uses to identify
// downloadable assets.
type Platform struct {
	OS   OS
	Arch Arch
}

// Slug formats the platform as "os-arch" (e.g. "macos-arm64"), matching the
// billing API path parameter used by GET /api/v1/downloads/{product}/{channel}/{slug}.
func (p Platform) Slug() string {
	return string(p.OS) + "-" + string(p.Arch)
}

// IsMacOS reports whether the platform targets macOS.
func (p Platform) IsMacOS() bool { return p.OS == OSMacOS }

// IsLinux reports whether the platform targets Linux.
func (p Platform) IsLinux() bool { return p.OS == OSLinux }

// IsWindows reports whether the platform targets Windows.
func (p Platform) IsWindows() bool { return p.OS == OSWindows }

// OSFromTarget normalizes a target triple OS component (the same values
// emitted by runtime.GOOS or Rust's std::env::consts::OS) to the billing OS
// identifier. Returns ("", false) for unsupported values.
func OSFromTarget(value string) (OS, bool) {
	switch strings.ToLower(value) {
	case "darwin", "macos":
		return OSMacOS, true
	case "linux":
		return OSLinux, true
	case "windows", "win32":
		return OSWindows, true
	}
	return "", false
}

// ArchFromTarget normalizes a target triple ARCH component (runtime.GOARCH or
// Rust's std::env::consts::ARCH) to the billing Arch identifier. Returns
// ("", false) for unsupported values.
func ArchFromTarget(value string) (Arch, bool) {
	switch strings.ToLower(value) {
	case "arm64", "aarch64":
		return ArchArm64, true
	case "amd64", "x86_64", "x64":
		return ArchX86_64, true
	}
	return "", false
}

// Detect returns the running host's Platform. The second return value is false
// on architectures or operating systems the billing API does not yet publish
// assets for.
func Detect() (Platform, bool) {
	os, ok := OSFromTarget(runtime.GOOS)
	if !ok {
		return Platform{}, false
	}
	arch, ok := ArchFromTarget(runtime.GOARCH)
	if !ok {
		return Platform{}, false
	}
	return Platform{OS: os, Arch: arch}, true
}

// DownloadURL builds the canonical billing download endpoint URL for the
// (product, channel, platform) triple. The base URL's trailing slash is
// normalized.
func DownloadURL(baseURL, product, channel string, platform Platform) string {
	base := strings.TrimRight(baseURL, "/")
	return base + "/api/v1/downloads/" + product + "/" + channel + "/" + platform.Slug()
}

// PickDownloadURL returns the download URL for the host platform, falling back
// to the provided platform when detection fails. Returns ("", false) when both
// detection and the fallback are absent.
func PickDownloadURL(baseURL, product, channel string, fallback *Platform) (string, bool) {
	p, ok := Detect()
	if !ok {
		if fallback == nil {
			return "", false
		}
		p = *fallback
	}
	return DownloadURL(baseURL, product, channel, p), true
}
