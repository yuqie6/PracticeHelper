package repo

import (
	"database/sql"
	"errors"
)

type Store struct {
	db *sql.DB
}

var ErrAlreadyImported = errors.New("project already imported")

func New(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Close() error {
	return s.db.Close()
}
