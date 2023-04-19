// Package app contains application instance.
package app

import (
	"github.com/Yandex-Practicum/go-musthave-shortener-trainer/internal/store"
)

// Instance describe app instance
type Instance struct {
	baseURL string

	store store.AuthStore
}

// NewInstance return new app instance.
func NewInstance(baseURL string, storage store.AuthStore) *Instance {
	return &Instance{
		baseURL: baseURL,
		store:   storage,
	}
}
