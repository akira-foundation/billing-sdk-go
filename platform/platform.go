// Package platform provides cross-environment platform detection matching the billing AssetPlatform slug.
package platform

import (
	"runtime"
	"strings"
)

type OS string
type Arch string

const (
	OSMacOS   OS = "macos"
	OSLinux   OS = "linux"
	OSWindows OS = "windows"

	ArchArm64  Arch = "arm64"
	ArchX86_64 Arch = "x86_64"
)

type Platform struct {
	OS   OS
	Arch Arch
}

func (p Platform) Slug() string {
	return string(p.OS) + "-" + string(p.Arch)
}

func (p Platform) IsMacOS() bool   { return p.OS == OSMacOS }
func (p Platform) IsLinux() bool   { return p.OS == OSLinux }
func (p Platform) IsWindows() bool { return p.OS == OSWindows }

// OSFromTarget accepts runtime.GOOS values and Rust std::env::consts::OS values.
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

// ArchFromTarget accepts runtime.GOARCH values and Rust std::env::consts::ARCH values.
func ArchFromTarget(value string) (Arch, bool) {
	switch strings.ToLower(value) {
	case "arm64", "aarch64":
		return ArchArm64, true
	case "amd64", "x86_64", "x64":
		return ArchX86_64, true
	}
	return "", false
}

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

func DownloadURL(baseURL, product, channel string, platform Platform) string {
	base := strings.TrimRight(baseURL, "/")
	return base + "/api/v1/downloads/" + product + "/" + channel + "/" + platform.Slug()
}

// PickDownloadURL falls back to the provided platform when detection fails.
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
