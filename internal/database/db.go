package database

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

type DB struct {
	conn *sql.DB
}

func New(dataDir string) (*DB, error) {
	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %v", err)
	}

	dbPath := filepath.Join(dataDir, "timetracker.db")
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	db := &DB{conn: conn}
	
	// Run migrations
	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %v", err)
	}

	return db, nil
}

func (db *DB) migrate() error {
	// Set up goose with embedded migrations
	goose.SetBaseFS(embedMigrations)
	
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("failed to set dialect: %v", err)
	}

	// Run migrations from embedded files
	if err := goose.Up(db.conn, "migrations"); err != nil {
		return fmt.Errorf("failed to run migrations: %v", err)
	}

	return nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

// Client operations
func (db *DB) GetClient(name string) (*Client, error) {
	var client Client
	err := db.conn.QueryRow(`
		SELECT id, name, rate, currency, active, notes 
		FROM clients WHERE name = ?
	`, name).Scan(&client.ID, &client.Name, &client.Rate, &client.Currency, &client.Active, &client.Notes)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &client, err
}

func (db *DB) CreateClient(client *Client) error {
	result, err := db.conn.Exec(`
		INSERT INTO clients (name, rate, currency, active, notes)
		VALUES (?, ?, ?, ?, ?)
	`, client.Name, client.Rate, client.Currency, client.Active, client.Notes)
	
	if err != nil {
		return err
	}
	
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	
	client.ID = id
	return nil
}

// Project operations
func (db *DB) GetProject(repoName string) (*Project, error) {
	var project Project
	err := db.conn.QueryRow(`
		SELECT id, client_id, repo_name, description, active
		FROM projects WHERE repo_name = ?
	`, repoName).Scan(&project.ID, &project.ClientID, &project.RepoName, &project.Description, &project.Active)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &project, err
}

func (db *DB) CreateProject(project *Project) error {
	result, err := db.conn.Exec(`
		INSERT INTO projects (client_id, repo_name, description, active)
		VALUES (?, ?, ?, ?)
	`, project.ClientID, project.RepoName, project.Description, project.Active)
	
	if err != nil {
		return err
	}
	
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	
	project.ID = id
	return nil
}

// Time entry operations
func (db *DB) CreateTimeEntry(entry *TimeEntry) error {
	result, err := db.conn.Exec(`
		INSERT INTO time_entries (project_id, date, hours, description, task_type, billable)
		VALUES (?, ?, ?, ?, ?, ?)
	`, entry.ProjectID, entry.Date, entry.Hours, entry.Description, entry.TaskType, entry.Billable)
	
	if err != nil {
		return err
	}
	
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	
	entry.ID = id
	return nil
}

// Types
type Client struct {
	ID       int64
	Name     string
	Rate     float64
	Currency string
	Active   bool
	Notes    sql.NullString
}

type Project struct {
	ID          int64
	ClientID    sql.NullInt64
	RepoName    string
	Description sql.NullString
	Active      bool
}

type TimeEntry struct {
	ID          int64
	ProjectID   int64
	Date        string
	Hours       float64
	Description sql.NullString
	TaskType    sql.NullString
	Billable    bool
	Billed      bool
	InvoiceID   sql.NullInt64
}