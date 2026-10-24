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
	b, err := io.ReadAll(CapReader(r, limit))
	if err != nil {
		return nil, err
	}
	return b, nil
}

// CapReader reads r up to limit bytes and fails with ErrTooLarge once r holds more, where io.LimitReader would end
// as if the input were complete.
func CapReader(r io.Reader, limit int64) io.Reader { return &capReader{r: r, left: limit} }

type capReader struct {
	r    io.Reader
	left int64
}

func (c *capReader) Read(p []byte) (int, error) {
	if c.left <= 0 {
		var probe [1]byte
		n, err := c.r.Read(probe[:])
		if n > 0 {
			return 0, ErrTooLarge
		}
		return 0, err
	}
	if int64(len(p)) > c.left {
		p = p[:c.left]
	}
	n, err := c.r.Read(p)
	c.left -= int64(n)
	return n, err
}
