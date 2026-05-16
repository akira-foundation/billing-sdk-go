package desktop

import (
	"crypto/sha256"
	"encoding/hex"
	"runtime"

	"github.com/denisbrodbeck/machineid"
)

type DeviceFingerprint struct {
	Fingerprint string `json:"fingerprint"`
	Platform    string `json:"platform"`
	AppVersion  string `json:"app_version"`
}

// DeviceFingerprintFor returns a stable per-machine fingerprint derived from
// the OS machine id, the runtime GOOS and the supplied app version.
func DeviceFingerprintFor(appVersion string) DeviceFingerprint {
	id, err := machineid.ID()
	if err != nil {
		id = "unknown"
	}
	h := sha256.New()
	h.Write([]byte(id))
	h.Write([]byte("::"))
	h.Write([]byte(runtime.GOOS))
	h.Write([]byte("::"))
	h.Write([]byte(appVersion))
	return DeviceFingerprint{
		Fingerprint: hex.EncodeToString(h.Sum(nil)),
		Platform:    runtime.GOOS,
		AppVersion:  appVersion,
	}
}
