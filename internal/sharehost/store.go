// Package sharehost is the hosted share short-code HTTP API.
package sharehost

import (
	"context"
	"errors"
	"time"
)

const (
	codeLen        = 10
	tokenBytes     = 32
	maxPayload     = 64 << 10
	createsPerHour = 20
	retainFor      = 365 * 24 * time.Hour
	lastReadGap    = 24 * time.Hour
)

var (
	ErrNotFound = errors.New("share not found")
	ErrExists   = errors.New("share exists")
)

// Meta is shares/<code>/meta.json. The owner token is never stored, only its SHA-256 hex.
type Meta struct {
	TokenHash string    `json:"tokenHash"`
	Latest    int       `json:"latest"`
	Created   time.Time `json:"created"`
	LastRead  time.Time `json:"lastRead,omitzero"`
}

// Version is shares/<code>/<n>.json.
type Version struct {
	Payload string `json:"payload"`
	Game    string `json:"game"`
}

// Store holds share codes. Implementations must not log credentials.
type Store interface {
	Create(ctx context.Context, code string, meta Meta, ver Version) error
	Meta(ctx context.Context, code string) (Meta, error)
	SaveMeta(ctx context.Context, code string, meta Meta) error
	SaveVersion(ctx context.Context, code string, n int, ver Version) error
	LoadVersion(ctx context.Context, code string, n int) (Version, error)
	Sweep(ctx context.Context, cutoff time.Time) (deleted, kept int, err error)
}

func ageRef(m Meta) time.Time {
	if !m.LastRead.IsZero() {
		return m.LastRead
	}
	return m.Created
}
