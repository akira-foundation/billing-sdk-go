// Package license owns the offline-snapshot helpers (decode, verify, remaining
// quota) and the authenticated license endpoint helpers (check, activate,
// refresh, sync-usage, public keys).
package license

// LicensingMode mirrors the products.licensing_mode column.
type LicensingMode string

const (
	LicensingModeOfflineSnapshot LicensingMode = "offline_snapshot"
	LicensingModeOnlineRealtime  LicensingMode = "online_realtime"
)

// UsagePeriod mirrors the plan_feature.usage_period enum.
type UsagePeriod string

const (
	UsagePeriodDaily   UsagePeriod = "daily"
	UsagePeriodWeekly  UsagePeriod = "weekly"
	UsagePeriodMonthly UsagePeriod = "monthly"
	UsagePeriodYearly  UsagePeriod = "yearly"
)

// UsageFeatureState is one entry of the snapshot's usage map. Type field
// switches the union: "bool" or "counter".
type UsageFeatureState struct {
	Type            string      `json:"type"`
	Enabled         bool        `json:"enabled,omitempty"`
	Allowance       uint64      `json:"allowance,omitempty"`
	Period          UsagePeriod `json:"period,omitempty"`
	PeriodStart     string      `json:"period_start,omitempty"`
	PeriodEnd       string      `json:"period_end,omitempty"`
	ConsumedAtIssue uint64      `json:"consumed_at_issue,omitempty"`
}

// SnapshotPayload is the decoded JSON of SignedLicense.Payload.
type SnapshotPayload struct {
	V                   int                          `json:"v,omitempty"`
	KeyID               string                       `json:"key_id"`
	CustomerID          string                       `json:"customer_id"`
	ProductKey          string                       `json:"product_key"`
	PlanKey             string                       `json:"plan_key"`
	LicensingMode       LicensingMode                `json:"licensing_mode,omitempty"`
	Features            map[string]bool              `json:"features"`
	Usage               map[string]UsageFeatureState `json:"usage,omitempty"`
	FingerprintHash     string                       `json:"fingerprint_hash"`
	Serial              uint64                       `json:"serial,omitempty"`
	IssuedAt            string                       `json:"issued_at"`
	ValidUntil          string                       `json:"valid_until"`
	PaidUpUntil         *string                      `json:"paid_up_until,omitempty"`
	FallbackReleaseDate *string                      `json:"fallback_release_date,omitempty"`
	UpdatesWindowDays   *uint32                      `json:"updates_window_days,omitempty"`
	OfflineGraceDays    *uint32                      `json:"offline_grace_days,omitempty"`
}

// SignedLicense carries the signed envelope returned by activate/refresh.
type SignedLicense struct {
	KeyID      string `json:"key_id"`
	Algorithm  string `json:"algorithm"`
	Payload    string `json:"payload"`
	Signature  string `json:"signature"`
	ValidUntil string `json:"valid_until"`
}

// ActivatedDevice describes the device slot returned with a license envelope.
type ActivatedDevice struct {
	ID         string `json:"id"`
	DeviceType string `json:"type"`
	SlotsUsed  int    `json:"slots_used"`
	SlotsLimit *int   `json:"slots_limit"`
}

// CheckPayload mirrors POST /api/licenses/check
type CheckPayload struct {
	Product string `json:"product"`
	Feature string `json:"feature"`
}

// CheckResponse describes a feature gate result.
type CheckResponse struct {
	Allowed bool    `json:"allowed"`
	Product string  `json:"product"`
	Plan    *string `json:"plan"`
	Feature string  `json:"feature"`
	Source  *string `json:"source"`
}

// ActivatePayload mirrors POST /api/licenses/activate
type ActivatePayload struct {
	Product     string  `json:"product"`
	DeviceType  string  `json:"device_type"`
	Platform    *string `json:"platform,omitempty"`
	DeviceName  *string `json:"device_name,omitempty"`
	AppVersion  *string `json:"app_version,omitempty"`
	Fingerprint string  `json:"fingerprint"`
}

// RefreshPayload mirrors POST /api/licenses/refresh
type RefreshPayload struct {
	Product     string `json:"product"`
	Fingerprint string `json:"fingerprint"`
}

// ActivateResponse mirrors the activate/refresh response.
type ActivateResponse struct {
	Allowed  bool            `json:"allowed"`
	Product  string          `json:"product"`
	Plan     string          `json:"plan"`
	Features map[string]bool `json:"features"`
	Device   ActivatedDevice `json:"device"`
	License  SignedLicense   `json:"license"`
}

// SyncUsagePayload mirrors POST /api/licenses/sync-usage
type SyncUsagePayload struct {
	Product     string            `json:"product"`
	Fingerprint string            `json:"fingerprint"`
	Serial      uint64            `json:"serial"`
	Deltas      map[string]uint64 `json:"deltas"`
}

// SyncUsageResponse carries the re-signed snapshot plus the applied deltas.
type SyncUsageResponse struct {
	License SignedLicense     `json:"license"`
	Applied map[string]uint64 `json:"applied"`
	Serial  uint64            `json:"serial"`
}

// PublicKey describes a registered Ed25519 verification key.
type PublicKey struct {
	KeyID           string `json:"key_id"`
	Algorithm       string `json:"algorithm"`
	PublicKeyBase64 string `json:"public_key_base64"`
}

// PublicKeysResponse mirrors GET /api/v1/license-keys/public
type PublicKeysResponse struct {
	Keys        []PublicKey `json:"keys"`
	ActiveKeyID *string     `json:"active_key_id"`
}
