package db

import (
	"database/sql"
	"fmt"
	"time"
)

// Crutch represents a running instance of an App (a Firecracker VM instance).
type Crutch struct {
	ID         string // UUID of this VM instance
	AppID      string // which app is running
	Pid        int    // firecracker process PID
	SocketPath string // firecracker control socket path
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// GetStateFsPath computes the state filesystem path from the VM instance ID.
// This ensures state_fs_path always matches the VM instance UUID.
// Returns: /var/lib/walkio/state/{id}.ext4
func (c *Crutch) GetStateFsPath() string {
	return fmt.Sprintf("/var/lib/walkio/state/%s.ext4", c.ID)
}

// InsertCrutch saves a new Crutch to the database.
func InsertCrutch(db *sql.DB, crutch *Crutch) error {
	query := `
		INSERT INTO crutches (id, app_id, pid, socket_path, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	now := time.Now().Unix()
	_, err := db.Exec(query,
		crutch.ID, crutch.AppID, crutch.Pid, crutch.SocketPath, now, now)
	return err
}

// GetCrutchByID retrieves a Crutch by ID from the database.
func GetCrutchByID(db *sql.DB, id string) (*Crutch, error) {
	query := `SELECT id, app_id, pid, socket_path, created_at, updated_at FROM crutches WHERE id = ?`
	row := db.QueryRow(query, id)

	var createdAt, updatedAt int64
	crutch := &Crutch{}
	err := row.Scan(&crutch.ID, &crutch.AppID, &crutch.Pid, &crutch.SocketPath,
		&createdAt, &updatedAt)

	if err != nil {
		return nil, err
	}

	crutch.CreatedAt = time.Unix(createdAt, 0)
	crutch.UpdatedAt = time.Unix(updatedAt, 0)
	return crutch, nil
}

// ListCrutchesByAppID retrieves all Crutches for an App from the database.
func ListCrutchesByAppID(db *sql.DB, appID string) ([]*Crutch, error) {
	query := `SELECT id, app_id, pid, socket_path, created_at, updated_at FROM crutches WHERE app_id = ? ORDER BY created_at DESC`
	rows, err := db.Query(query, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var crutches []*Crutch
	for rows.Next() {
		var createdAt, updatedAt int64
		crutch := &Crutch{}
		if err := rows.Scan(&crutch.ID, &crutch.AppID, &crutch.Pid, &crutch.SocketPath,
			&createdAt, &updatedAt); err != nil {
			return nil, err
		}
		crutch.CreatedAt = time.Unix(createdAt, 0)
		crutch.UpdatedAt = time.Unix(updatedAt, 0)
		crutches = append(crutches, crutch)
	}

	return crutches, rows.Err()
}

// DeleteCrutch removes a Crutch from the database.
func DeleteCrutch(db *sql.DB, id string) error {
	query := `DELETE FROM crutches WHERE id = ?`
	_, err := db.Exec(query, id)
	return err
}
