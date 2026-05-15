package billing

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"math"
	"testing"
	"time"
)

func basePayload() LicenseSnapshotPayload {
	paidUp := "2027-05-15T00:00:00Z"
	fallback := "2027-05-15T00:00:00Z"
	return LicenseSnapshotPayload{
		V:             2,
		KeyID:         "k1",
		CustomerID:    "cust-1",
		ProductKey:    "maintainer",
		PlanKey:       "free",
		LicensingMode: LicensingModeOfflineSnapshot,
		Features:      map[string]bool{"agent_run": true},
		Usage: map[string]UsageFeatureState{
			"agent_run": {
				Type:            "counter",
				Allowance:       5,
				Period:          UsagePeriodMonthly,
				PeriodStart:     "2026-05-01T00:00:00Z",
				PeriodEnd:       "2026-05-31T00:00:00Z",
				ConsumedAtIssue: 2,
			},
		},
		FingerprintHash:     "fp",
		Serial:              1,
		IssuedAt:            "2026-05-15T10:00:00Z",
		ValidUntil:          "2026-05-29T10:00:00Z",
		PaidUpUntil:         &paidUp,
		FallbackReleaseDate: &fallback,
	}
}

func signPayload(t *testing.T, payload LicenseSnapshotPayload, priv ed25519.PrivateKey) SignedLicense {
	t.Helper()
	bytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	sig := ed25519.Sign(priv, bytes)
	return SignedLicense{
		KeyID:      payload.KeyID,
		Algorithm:  "ed25519",
		Payload:    base64.StdEncoding.EncodeToString(bytes),
		Signature:  base64.StdEncoding.EncodeToString(sig),
		ValidUntil: payload.ValidUntil,
	}
}

func TestDecodeLicense(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	_ = pub
	signed := signPayload(t, basePayload(), priv)
	decoded, err := DecodeLicense(signed)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded.Payload.PlanKey != "free" {
		t.Fatalf("plan_key: got %q", decoded.Payload.PlanKey)
	}
	if decoded.Payload.Serial != 1 {
		t.Fatalf("serial: got %d", decoded.Payload.Serial)
	}
}

func TestVerifyLicenseRoundtrip(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	pubB64 := base64.StdEncoding.EncodeToString(pub)
	signed := signPayload(t, basePayload(), priv)
	ok, err := VerifyLicense(signed, pubB64)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !ok {
		t.Fatalf("expected valid")
	}
}

func TestVerifyLicenseRejectsWrongKey(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	wrong, _, _ := ed25519.GenerateKey(nil)
	signed := signPayload(t, basePayload(), priv)
	ok, err := VerifyLicense(signed, base64.StdEncoding.EncodeToString(wrong))
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if ok {
		t.Fatalf("expected invalid")
	}
}

func TestVerifyLicenseRejectsNonEd25519(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	signed := signPayload(t, basePayload(), priv)
	signed.Algorithm = "rsa"
	ok, _ := VerifyLicense(signed, "AAAA")
	if ok {
		t.Fatalf("expected reject")
	}
}

func TestComputeRemaining(t *testing.T) {
	p := basePayload()

	remaining, unlim, ok := ComputeRemaining(p, "agent_run", 0)
	if !ok || unlim || remaining != 3 {
		t.Fatalf("got (%d,%v,%v) want (3,false,true)", remaining, unlim, ok)
	}

	remaining, _, ok = ComputeRemaining(p, "agent_run", 2)
	if !ok || remaining != 1 {
		t.Fatalf("got %d want 1", remaining)
	}

	remaining, _, ok = ComputeRemaining(p, "agent_run", 100)
	if !ok || remaining != 0 {
		t.Fatalf("got %d want 0", remaining)
	}

	_, _, ok = ComputeRemaining(p, "ghost", 0)
	if ok {
		t.Fatalf("expected ok=false for unknown feature")
	}
}

func TestComputeRemainingBool(t *testing.T) {
	p := basePayload()
	p.Usage = map[string]UsageFeatureState{
		"white_label": {Type: "bool", Enabled: true},
	}
	remaining, unlim, ok := ComputeRemaining(p, "white_label", 0)
	if !ok || !unlim || remaining != math.MaxUint64 {
		t.Fatalf("got (%d,%v,%v) want unlim", remaining, unlim, ok)
	}

	p.Usage["white_label"] = UsageFeatureState{Type: "bool", Enabled: false}
	remaining, unlim, ok = ComputeRemaining(p, "white_label", 0)
	if !ok || unlim || remaining != 0 {
		t.Fatalf("got (%d,%v,%v) want disabled", remaining, unlim, ok)
	}
}

func TestExpiryAndGrace(t *testing.T) {
	p := basePayload()
	now := time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC)
	after := time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC)

	if IsExpired(p, now) {
		t.Fatalf("expected not expired at %v", now)
	}
	if !IsExpired(p, after) {
		t.Fatalf("expected expired at %v", after)
	}

	graceWindow := int64(7 * 24 * 3600)
	if !IsInGrace(p, graceWindow, time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("expected in grace")
	}
	if IsInGrace(p, graceWindow, time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("expected out of grace")
	}
}

func TestCanUseUpdate(t *testing.T) {
	p := basePayload()
	if !CanUseUpdate(p, time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("should allow")
	}
	if CanUseUpdate(p, time.Date(2028, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("should block")
	}

	p.PaidUpUntil = nil
	if !CanUseUpdate(p, time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("nil paid_up_until should allow all")
	}
}

func TestPeriodResetAt(t *testing.T) {
	p := basePayload()
	reset := PeriodResetAt(p, "agent_run")
	if reset.IsZero() {
		t.Fatalf("expected period_end")
	}
	if reset.Year() != 2026 || reset.Month() != 5 || reset.Day() != 31 {
		t.Fatalf("got %v", reset)
	}

	p.Usage = map[string]UsageFeatureState{"x": {Type: "bool", Enabled: true}}
	if !PeriodResetAt(p, "x").IsZero() {
		t.Fatalf("bool should return zero")
	}
}
