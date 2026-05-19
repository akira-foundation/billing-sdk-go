package desktop

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

// CheckoutURL builds the public checkout URL for a product on the billing site.
func CheckoutURL(baseURL, product string) string {
	return fmt.Sprintf("%s/plans?product=%s", strings.TrimRight(baseURL, "/"), url.QueryEscape(product))
}

// EnvSpec describes an environment-driven configuration value. Release builds
// fall back to the BakedDefault (typically wired via ldflags / build tags);
// debug-style consumers can opt in to runtime overrides by setting Overridable
// to true.
type EnvSpec struct {
	Name          string
	BakedDefault  string
	Overridable   bool
	FallbackValue string
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
