package fsx

import (
	"errors"
	"io"
)

// ErrTooLarge is what ReadCapped returns for input over its limit.
var ErrTooLarge = errors.New("larger than its size cap")

// ReadCapped reads r whole, failing with ErrTooLarge when it holds more than limit bytes. It is the bound on every
// read of input Mortar did not write: at most limit+1 bytes are ever held.
func ReadCapped(r io.Reader, limit int64) ([]byte, error) {
	b, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, ErrTooLarge
	}
	return b, nil
}
