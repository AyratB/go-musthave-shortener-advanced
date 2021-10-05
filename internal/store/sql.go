package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/jackc/pgx/pgtype"
)

var _ Store = (*RDB)(nil)
var _ AuthStore = (*RDB)(nil)

type RDB struct {
	db *sql.DB
}

func NewRDB(db *sql.DB) *RDB {
	return &RDB{
		db: db,
	}
}

// Bootstrap creates all necessary tables and their structures
func (r *RDB) Bootstrap(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS urls (
			id serial PRIMARY KEY,
			original_url text,
			user_id uuid,
			updated_at timestamp without time zone,
		    deleted_at timestamp without time zone
		);

		CREATE INDEX IF NOT EXISTS user_id_idx ON urls (user_id);
		CREATE UNIQUE INDEX IF NOT EXISTS original_url_idx ON urls (original_url) WHERE deleted_at IS NULL;
	`

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("cannot start transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("cannot create `urls` table: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("cannot commit transaction: %w", err)
	}
	return nil
}

func (r *RDB) Save(ctx context.Context, url *url.URL) (id string, err error) {
	query := `
		INSERT INTO urls
		    (original_url)
		VALUES
		    ($1)
		ON CONFLICT (original_url) WHERE deleted_at IS NULL
		DO UPDATE SET updated_at = NOW()
		RETURNING
		    id,
		    updated_at
	`

	var lid int64
	var updatedAt *time.Time
	err = r.db.QueryRowContext(ctx, query, url.String()).Scan(&lid, &updatedAt)
	if err != nil {
		return "", fmt.Errorf("cannot fetch conflict url: %w", err)
	}

	id = strconv.FormatInt(lid, 10)
	if updatedAt != nil && !updatedAt.IsZero() {
		err = ErrConflict
	}
	return
}

func (r *RDB) SaveBatch(ctx context.Context, urls []*url.URL) (ids []string, err error) {
	args := make([]interface{}, 0, len(urls))

	var insertValues strings.Builder
	for i, u := range urls {
		if i > 0 {
			insertValues.WriteByte(',')
		}
		insertValues.WriteString("($" + strconv.Itoa(i+1) + ")")
		args = append(args, u.String())
	}

	query := `
		INSERT INTO urls
			(original_url)
		VALUES ` + insertValues.String() + `
		ON CONFLICT (original_url) WHERE deleted_at IS NULL
		DO UPDATE SET updated_at = NOW()
		RETURNING id
	`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}
		ids = append(ids, strconv.FormatInt(id, 10))
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	if len(ids) != len(urls) {
		return nil, errors.New("not all URLs have been saved")
	}

	return ids, nil
}

func (r *RDB) Load(ctx context.Context, id string) (url *url.URL, err error) {
	var rawURL string
	var deletedAt *time.Time
	query := `SELECT original_url, deleted_at FROM urls WHERE id = $1;`

	err = r.db.QueryRowContext(ctx, query, id).Scan(&rawURL, &deletedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("cannot scan row: %w", err)
	}
	if deletedAt != nil {
		return nil, ErrDeleted
	}

	return url.Parse(rawURL)
}

func (r *RDB) SaveUser(ctx context.Context, uid uuid.UUID, url *url.URL) (id string, err error) {
	query := `
		INSERT INTO urls
		    (original_url, user_id)
		VALUES
		    ($1, $2)
		ON CONFLICT (original_url) WHERE deleted_at IS NULL
		DO UPDATE SET updated_at = NOW()
		RETURNING
		    id,
		    updated_at
	`

	var lid int64
	var updatedAt *time.Time
	err = r.db.QueryRowContext(ctx, query, url.String(), uid).Scan(&lid, &updatedAt)
	if err != nil {
		return "", fmt.Errorf("cannot fetch conflict url: %w", err)
	}

	id = strconv.FormatInt(lid, 10)
	if updatedAt != nil && !updatedAt.IsZero() {
		err = ErrConflict
	}
	return
}

func (r *RDB) SaveUserBatch(ctx context.Context, uid uuid.UUID, urls []*url.URL) (ids []string, err error) {
	args := make([]interface{}, 0, len(urls)+1)
	uidPos := len(urls) + 1

	var insertValues strings.Builder
	for i, u := range urls {
		if i > 0 {
			insertValues.WriteByte(',')
		}
		insertValues.WriteString("($" + strconv.Itoa(i+1) + ", $" + strconv.Itoa(uidPos) + ")")
		args = append(args, u.String())
	}
	args = append(args, uid)

	query := `INSERT INTO urls (original_url, user_id) VALUES ` + insertValues.String() + ` RETURNING id;`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}
		ids = append(ids, strconv.FormatInt(id, 10))
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	if len(ids) != len(urls) {
		return nil, errors.New("not all URLs have been saved")
	}

	return ids, nil
}

func (r *RDB) LoadUser(ctx context.Context, uid uuid.UUID, id string) (url *url.URL, err error) {
	var rawURL string
	var deletedAt *time.Time
	query := `SELECT original_url, deleted_at FROM urls WHERE id = $1 AND user_id = $2;`

	err = r.db.QueryRowContext(ctx, query, id, uid).Scan(&rawURL, &deletedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("cannot scan row: %w", err)
	}
	if deletedAt != nil {
		return nil, ErrDeleted
	}

	return url.Parse(rawURL)
}

func (r *RDB) LoadUsers(ctx context.Context, uid uuid.UUID) (urls map[string]*url.URL, err error) {
	query := `SELECT id, original_url FROM urls WHERE user_id = $1 AND deleted_at IS NULL;`

	rows, err := r.db.QueryContext(ctx, query, uid)
	if err != nil {
		return nil, fmt.Errorf("cannot query rows: %w", err)
	}
	defer rows.Close()

	res := make(map[string]*url.URL)
	for rows.Next() {
		var id int64
		var rawURL string

		if err := rows.Scan(&id, &rawURL); err != nil {
			return nil, fmt.Errorf("cannot scan row: %w", err)
		}
		u, err := url.Parse(rawURL)
		if err != nil {
			return nil, fmt.Errorf("cannot parse URL: %w", err)
		}

		res[strconv.FormatInt(id, 10)] = u
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return res, nil
}

func (r *RDB) DeleteUsers(ctx context.Context, uid uuid.UUID, ids ...string) error {
	arr := new(pgtype.VarcharArray)
	if err := arr.Set(ids); err != nil {
		return fmt.Errorf("cannot set ids to pg variable: %w", err)
	}

	query := `UPDATE urls SET deleted_at = NOW() WHERE user_id = $1 AND id = ANY($2);`
	_, err := r.db.ExecContext(ctx, query, uid, arr)
	return err
}

func (r *RDB) Ping(ctx context.Context) error {
	return r.db.PingContext(ctx)
}

func (r *RDB) Close() error {
	return r.db.Close()
}
