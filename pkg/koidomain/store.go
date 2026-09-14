package koidomain

import "context"

// Store aggregates persistence. Add repo getters as entities land (Brand in slice 1).
type Store interface {
	Atomic(context.Context, func(Store) error) error
	Ping(context.Context) error
}
