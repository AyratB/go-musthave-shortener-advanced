package store

import (
	"context"
	"errors"
	"io"
	"net/url"

	"github.com/gofrs/uuid"
)

var (
	ErrDeleted = errors.New("record deleted")
)

// Store interface
type Store interface {
	io.Closer

	Save(ctx context.Context, url *url.URL) (id string, err error)
	Load(ctx context.Context, id string) (url *url.URL, err error)
	Ping(ctx context.Context) error
}

// BatchStore interface
type BatchStore interface {
	Store

	SaveBatch(ctx context.Context, urls []*url.URL) (ids []string, err error)
}

// AuthStore interface
type AuthStore interface {
	BatchStore

	SaveUser(ctx context.Context, uid uuid.UUID, url *url.URL) (id string, err error)
	SaveUserBatch(ctx context.Context, uid uuid.UUID, urls []*url.URL) (ids []string, err error)
	LoadUser(ctx context.Context, uid uuid.UUID, id string) (url *url.URL, err error)
	LoadUsers(ctx context.Context, uid uuid.UUID) (urls map[string]*url.URL, err error)
	DeleteUsers(ctx context.Context, uid uuid.UUID, ids ...string) error
}
