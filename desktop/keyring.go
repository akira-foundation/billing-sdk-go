package desktop

import (
	"errors"

	"github.com/zalando/go-keyring"
)

type TokenKeyring struct {
	Service string
	Account string
}

func NewTokenKeyring(service, account string) TokenKeyring {
	return TokenKeyring{Service: service, Account: account}
}

func (k TokenKeyring) Get() (string, bool, error) {
	v, err := keyring.Get(k.Service, k.Account)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return v, true, nil
}

func (k TokenKeyring) Set(value string) error {
	return keyring.Set(k.Service, k.Account, value)
}

func (k TokenKeyring) Delete() error {
	err := keyring.Delete(k.Service, k.Account)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}
