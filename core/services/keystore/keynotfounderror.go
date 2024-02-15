package keystore

import "fmt"

// KeyNotFoundError is returned when we don't find a requested key
type KeyNotFoundError struct {
	ID      string
	KeyType string
}

func (e KeyNotFoundError) Error() string {
	return fmt.Sprintf("unable to find %s key with id %s", e.KeyType, e.ID)
}
