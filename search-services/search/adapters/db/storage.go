package db

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/liy0aay/xkcd-search/search/core"
)

type DB struct {
	log  *slog.Logger
	conn *sqlx.DB
}

func New(log *slog.Logger, address string) (*DB, error) {

	db, err := sqlx.Connect("pgx", address)
	if err != nil {
		log.Error("connection problem", "address", address, "error", err)
		return nil, err
	}

	return &DB{
		log:  log,
		conn: db,
	}, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) Search(ctx context.Context, keyword string) ([]int, error) {
	var IDs []int
	err := db.conn.SelectContext(
		ctx, &IDs,
		"SELECT id FROM comics WHERE $1 = ANY(words)",
		keyword,
	)

	return IDs, err
}

type Comics struct {
	ID       int            `db:"id"`
	URL      string         `db:"url"`
	Title    string         `db:"title"`
	Alt      string         `db:"alt"`
	Keywords pq.StringArray `db:"words"`
}

func (db *DB) Get(ctx context.Context, id int) (core.Comics, error) {
	var comics Comics
	err := db.conn.GetContext(
		ctx, &comics,
		"SELECT id, url, title, alt, words FROM comics WHERE id = $1",
		id,
	)
	if errors.Is(err, sql.ErrNoRows) {
		err = core.ErrNotFound
	}

	return core.Comics{ID: comics.ID, URL: comics.URL, Title: comics.Title, Alt: comics.Alt, Keywords: comics.Keywords}, err
}

func (db *DB) LastID(ctx context.Context) (int, error) {
	var ID int
	err := db.conn.GetContext(
		ctx, &ID,
		"SELECT coalesce(max(id), 0) FROM comics",
	)

	return ID, err
}
