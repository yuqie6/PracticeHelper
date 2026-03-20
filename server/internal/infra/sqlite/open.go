package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

func Open(databasePath string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(databasePath), 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	dsn := fmt.Sprintf("file:%s?_busy_timeout=5000&_foreign_keys=on", databasePath)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	return db, nil
}
