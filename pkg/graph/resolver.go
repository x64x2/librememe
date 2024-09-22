package graph

import (
	"time"

	"github.com/google/uuid"
)

//go:generate go run github.com/99designs/gqlgen generate

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	profileDataLoader *ProfileLoader
}

func NewResolver() (*Resolver, error) {
	r := &Resolver{}
	r.profileDataLoader = &ProfileLoader{
		wait:     2 * time.Millisecond,
		maxBatch: 100,
		fetch:    r.profileDataloaderFn,
	}
	return r, nil
}

func IsValidUUID(u uuid.UUID) bool {
	return u != uuid.Nil && u.Version() == 4
}
