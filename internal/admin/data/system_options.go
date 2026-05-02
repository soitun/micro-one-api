package data

import (
	"database/sql"
	"fmt"
	"time"
)

// SystemOption represents a system configuration entry.
type SystemOption struct {
	Key   string
	Value string
}

// SystemOptionsRepo provides CRUD for system_options table.
type SystemOptionsRepo struct {
	db *sql.DB
}

// NewSystemOptionsRepo creates a new repo from a database connection.
func NewSystemOptionsRepo(db *sql.DB) *SystemOptionsRepo {
	return &SystemOptionsRepo{db: db}
}

// Get returns the value for a given key, or empty string if not found.
func (r *SystemOptionsRepo) Get(key string) (string, error) {
	var value string
	err := r.db.QueryRow("SELECT option_value FROM system_options WHERE option_key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get system option %s: %w", key, err)
	}
	return value, nil
}

// Set upserts a key-value pair.
func (r *SystemOptionsRepo) Set(key, value string) error {
	_, err := r.db.Exec(
		"INSERT INTO system_options (option_key, option_value, updated_at) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE option_value = VALUES(option_value), updated_at = VALUES(updated_at)",
		key, value, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("set system option %s: %w", key, err)
	}
	return nil
}

// GetAll returns all system options.
func (r *SystemOptionsRepo) GetAll() ([]SystemOption, error) {
	rows, err := r.db.Query("SELECT option_key, option_value FROM system_options")
	if err != nil {
		return nil, fmt.Errorf("list system options: %w", err)
	}
	defer rows.Close()

	var opts []SystemOption
	for rows.Next() {
		var o SystemOption
		if err := rows.Scan(&o.Key, &o.Value); err != nil {
			return nil, fmt.Errorf("scan system option: %w", err)
		}
		opts = append(opts, o)
	}
	return opts, rows.Err()
}
