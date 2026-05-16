package desktop

import "os"

// EnvSpec describes an environment-driven configuration value. Release builds
// fall back to the BakedDefault (typically wired via ldflags / build tags);
// debug-style consumers can opt in to runtime overrides by setting Overridable
// to true.
type EnvSpec struct {
	Name           string
	BakedDefault   string
	Overridable    bool
	FallbackValue  string
}

func Resolve(spec EnvSpec) string {
	if spec.Overridable {
		if v := os.Getenv(spec.Name); v != "" {
			return v
		}
	}
	if spec.BakedDefault != "" {
		return spec.BakedDefault
	}
	return spec.FallbackValue
}
