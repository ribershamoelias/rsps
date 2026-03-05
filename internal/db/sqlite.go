package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

func OpenSQLite(path string) (*sql.DB, error) {
	connection, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database '%s': %w", path, err)
	}

	if _, err := connection.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		connection.Close()
		return nil, fmt.Errorf("enable foreign_keys pragma: %w", err)
	}

	if _, err := connection.Exec(`PRAGMA busy_timeout = 5000;`); err != nil {
		connection.Close()
		return nil, fmt.Errorf("set busy_timeout pragma: %w", err)
	}

	return connection, nil
}
