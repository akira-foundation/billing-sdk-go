package desktop

import (
	billing "github.com/akira-io/billing-sdk-go"
)

// SessionStore binds an SDK Client to an OS keychain entry. Apps call Hydrate
// at boot, Persist after sign-in and Clear on logout.
type SessionStore struct {
	keyring TokenKeyring
}

func NewSessionStore(k TokenKeyring) *SessionStore {
	return &SessionStore{keyring: k}
}

// Hydrate copies any saved token into client. Returns true when a token was
// found and applied.
func (s *SessionStore) Hydrate(client *billing.Client) (bool, error) {
	v, ok, err := s.keyring.Get()
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	client.SetCustomerToken(v)
	return true, nil
}

func (s *SessionStore) Persist(client *billing.Client, token string) error {
	if err := s.keyring.Set(token); err != nil {
		return err
	}
	client.SetCustomerToken(token)
	return nil
}

func (s *SessionStore) Clear(client *billing.Client) error {
	if err := s.keyring.Delete(); err != nil {
		return err
	}
	client.SetCustomerToken("")
	return nil
}

func (s *SessionStore) HasToken() bool {
	_, ok, _ := s.keyring.Get()
	return ok
}
