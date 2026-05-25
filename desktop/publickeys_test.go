package desktop

import (
	"errors"
	"testing"

	"github.com/akira-io/billing-sdk-go/license"
)

func TestParsePublicKeyStore_KeyedAndBare(t *testing.T) {
	store := ParsePublicKeyStore("k1:AAA, BBB ,k2:CCC")
	snap := store.Snapshot()
	if snap["k1"] != "AAA" {
		t.Fatalf("k1 = %q, want AAA", snap["k1"])
	}
	if snap["k2"] != "CCC" {
		t.Fatalf("k2 = %q, want CCC", snap["k2"])
	}
	if snap["default"] != "BBB" {
		t.Fatalf("default = %q, want BBB", snap["default"])
	}
}

func TestPublicKeyStore_MergeKeys(t *testing.T) {
	store := NewPublicKeyStore()
	store.MergeKeys([]license.PublicKey{
		{KeyID: "k1", PublicKeyBase64: "AAA"},
		{KeyID: "", PublicKeyBase64: "skip"},
	})
	if store.Len() != 1 {
		t.Fatalf("len = %d, want 1", store.Len())
	}
	if store.Snapshot()["k1"] != "AAA" {
		t.Fatalf("k1 not merged")
	}
}

func TestIsUnknownSigningKey(t *testing.T) {
	if !isUnknownSigningKey(errors.New("unknown signing key_id: k9")) {
		t.Fatal("expected match")
	}
	if isUnknownSigningKey(errors.New("network down")) {
		t.Fatal("unexpected match")
	}
}
