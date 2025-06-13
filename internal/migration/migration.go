package migration

import (
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// Migration represents a database migration
type Migration struct {
	Version string
	Path    string
	Content string
}

// Runner handles database migrations
type Runner struct {
	db *sql.DB
}

// NewRunner creates a new migration runner
func NewRunner(db *sql.DB) *Runner {
	return &Runner{db: db}
}

// Run executes all pending migrations
func (r *Runner) Run(migrationsDir string) error {
	// Ensure migrations table exists
	if err := r.createMigrationsTable(); err != nil {
		return fmt.Errorf("failed to create migrations table: %v", err)
	}

	// Get all migration files
	migrations, err := r.loadMigrations(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to load migrations: %v", err)
	}

	// Get applied migrations
	appliedMigrations, err := r.getAppliedMigrations()
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %v", err)
	}

	// Apply pending migrations
	for _, migration := range migrations {
		if r.isMigrationApplied(migration.Version, appliedMigrations) {
			fmt.Printf("Migration %s already applied, skipping\n", migration.Version)
			continue
		}

		fmt.Printf("Applying migration %s...\n", migration.Version)
		if err := r.applyMigration(migration); err != nil {
			return fmt.Errorf("failed to apply migration %s: %v", migration.Version, err)
		}
		fmt.Printf("Migration %s applied successfully\n", migration.Version)
	}

	return nil
}

// createMigrationsTable creates the migrations tracking table
func (r *Runner) createMigrationsTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS migrations (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			version VARCHAR(255) NOT NULL UNIQUE,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`
	_, err := r.db.Exec(query)
	return err
}

// loadMigrations loads all migration files from the directory
func (r *Runner) loadMigrations(migrationsDir string) ([]*Migration, error) {
	var migrations []*Migration

	err := filepath.WalkDir(migrationsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".sql") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %v", path, err)
		}

		// Extract version from filename (e.g., "001_initial_schema.sql" -> "001")
		filename := filepath.Base(path)
		version := strings.Split(filename, "_")[0]

		migrations = append(migrations, &Migration{
			Version: version,
			Path:    path,
			Content: string(content),
		})

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort migrations by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// getAppliedMigrations returns a list of applied migration versions
func (r *Runner) getAppliedMigrations() (map[string]bool, error) {
	query := "SELECT version FROM migrations"
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}

	return applied, nil
}

// isMigrationApplied checks if a migration has been applied
func (r *Runner) isMigrationApplied(version string, appliedMigrations map[string]bool) bool {
	return appliedMigrations[version]
}

// applyMigration applies a single migration
func (r *Runner) applyMigration(migration *Migration) error {
	// Start transaction
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Split SQL statements and execute them individually
	statements := r.splitSQLStatements(migration.Content)
	for i, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(stmt, "--") {
			continue // Skip empty lines and comments
		}

		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("failed to execute migration SQL statement %d: %v\nStatement: %s", i+1, err, stmt)
		}
	}

	// Record migration as applied
	query := "INSERT INTO migrations (version, applied_at) VALUES (?, ?)"
	if _, err := tx.Exec(query, migration.Version, time.Now()); err != nil {
		return fmt.Errorf("failed to record migration: %v", err)
	}

	// Commit transaction
	return tx.Commit()
}

// splitSQLStatements splits a SQL string into individual statements
func (r *Runner) splitSQLStatements(sql string) []string {
	// Split by semicolon, but be careful about semicolons in strings
	var statements []string
	var current strings.Builder
	inString := false
	var stringChar rune

	runes := []rune(sql)
	for i, r := range runes {
		switch r {
		case '\'', '"', '`':
			if !inString {
				inString = true
				stringChar = r
			} else if r == stringChar {
				// Check if it's escaped
				if i > 0 && runes[i-1] != '\\' {
					inString = false
				}
			}
			current.WriteRune(r)
		case ';':
			current.WriteRune(r)
			if !inString {
				stmt := strings.TrimSpace(current.String())
				if stmt != "" {
					statements = append(statements, stmt)
				}
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}

	// Add the last statement if it doesn't end with semicolon
	if current.Len() > 0 {
		stmt := strings.TrimSpace(current.String())
		if stmt != "" {
			statements = append(statements, stmt)
		}
	}

	return statements
}

// Status shows the status of all migrations
func (r *Runner) Status(migrationsDir string) error {
	// Ensure migrations table exists
	if err := r.createMigrationsTable(); err != nil {
		return fmt.Errorf("failed to create migrations table: %v", err)
	}

	// Get all migration files
	migrations, err := r.loadMigrations(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to load migrations: %v", err)
	}

	// Get applied migrations
	appliedMigrations, err := r.getAppliedMigrations()
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %v", err)
	}

	fmt.Println("Migration Status:")
	fmt.Println("================")
	for _, migration := range migrations {
		status := "PENDING"
		if r.isMigrationApplied(migration.Version, appliedMigrations) {
			status = "APPLIED"
		}
		fmt.Printf("%s: %s\n", migration.Version, status)
	}

	return nil
}
