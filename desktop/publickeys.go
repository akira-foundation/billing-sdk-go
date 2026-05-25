package desktop

import (
	"context"
	"strings"
	"sync"

	"github.com/akira-io/billing-sdk-go/client"
	"github.com/akira-io/billing-sdk-go/license"
)

// PublicKeyStore holds license signing public keys indexed by key_id. It seeds
// from a baked key string, merges keys fetched from the API, and exposes a
// snapshot suitable for ActivateOrRefresh.
type PublicKeyStore struct {
	mu   sync.RWMutex
	keys map[string]string
}

// NewPublicKeyStore returns an empty store.
func NewPublicKeyStore() *PublicKeyStore {
	return &PublicKeyStore{keys: make(map[string]string)}
}

// ParsePublicKeyStore seeds a store from a baked "key_id:base64,key_id:base64"
// string. An entry without a colon is stored under the "default" key id.
func ParsePublicKeyStore(baked string) *PublicKeyStore {
	store := NewPublicKeyStore()
	for _, entry := range strings.Split(baked, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		keyID, encoded := "default", entry
		if idx := strings.Index(entry, ":"); idx >= 0 {
			keyID = strings.TrimSpace(entry[:idx])
			encoded = strings.TrimSpace(entry[idx+1:])
		}
		if encoded != "" {
			store.keys[keyID] = encoded
		}
	}
	return store
}

// Snapshot returns a copy of the current key_id to base64 map.
func (s *PublicKeyStore) Snapshot() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]string, len(s.keys))
	for id, encoded := range s.keys {
		out[id] = encoded
	}
	return out
}

// MergeKeys adds or replaces keys returned by license.PublicKeys.
func (s *PublicKeyStore) MergeKeys(keys []license.PublicKey) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, k := range keys {
		if k.KeyID != "" && k.PublicKeyBase64 != "" {
			s.keys[k.KeyID] = k.PublicKeyBase64
		}
	}
}

// Len reports how many keys the store holds.
func (s *PublicKeyStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.keys)
}

// ActivateOrRefreshWithKeys runs ActivateOrRefresh against the store. On an
// unknown signing key_id it fetches the latest public keys, merges them, and
// retries once. This removes the per-application key cache and rotation
// handling each desktop consumer would otherwise reimplement.
func ActivateOrRefreshWithKeys(ctx context.Context, c *client.Client, opts ActivateOrRefreshOptions, store *PublicKeyStore) (*VerifiedLicense, error) {
	verified, err := ActivateOrRefresh(ctx, c, opts, store.Snapshot())
	if err == nil || !isUnknownSigningKey(err) {
		return verified, err
	}
	resp, refreshErr := license.PublicKeys(ctx, c)
	if refreshErr != nil {
		return nil, err
	}
	store.MergeKeys(resp.Keys)
	return ActivateOrRefresh(ctx, c, opts, store.Snapshot())
}

func isUnknownSigningKey(err error) bool {
	return err != nil && strings.Contains(err.Error(), "unknown signing key_id")
}
