package desktop

import (
	"github.com/akira-io/billing-sdk-go/client"
)

// SessionStore binds an SDK Client to an OS keychain entry. Apps call Hydrate
// at boot, Persist after sign-in and Clear on logout.
type SessionStore struct {
	keyring TokenKeyring
}

func NewSessionStore(k TokenKeyring) *SessionStore {
	return &SessionStore{keyring: k}
}

// Hydrate copies any saved token into c. Returns true when a token was
// found and applied.
func (s *SessionStore) Hydrate(c *client.Client) (bool, error) {
	v, ok, err := s.keyring.Get()
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	c.SetCustomerToken(v)
	return true, nil
}

func (s *SessionStore) Persist(c *client.Client, token string) error {
	if err := s.keyring.Set(token); err != nil {
		return err
	}
	c.SetCustomerToken(token)
	return nil
}

func (s *SessionStore) Clear(c *client.Client) error {
	if err := s.keyring.Delete(); err != nil {
		return err
	}
	c.SetCustomerToken("")
	return nil
}

func (s *SessionStore) HasToken() bool {
	_, ok, _ := s.keyring.Get()
	return ok
}
