package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return db, nil
}

const schemaJobs = `CREATE TABLE IF NOT EXISTS jobs (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	description TEXT DEFAULT '',
	cron_expression TEXT NOT NULL,
	python_code TEXT NOT NULL,
	image_id INTEGER NOT NULL REFERENCES runner_images(id),
	env_vars TEXT DEFAULT '{}',
	host_mappings TEXT DEFAULT '[]',
	log_level TEXT DEFAULT 'info',
	enabled BOOLEAN DEFAULT 1,
	created_by INTEGER REFERENCES users(id),
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`

func Migrate(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		role TEXT NOT NULL DEFAULT 'user',
		must_change_password INTEGER NOT NULL DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	` + schemaJobs + `

	CREATE TABLE IF NOT EXISTS executions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		job_id INTEGER REFERENCES jobs(id) ON DELETE CASCADE,
		status TEXT NOT NULL DEFAULT 'running',
		stdout TEXT DEFAULT '',
		stderr TEXT DEFAULT '',
		exit_code INTEGER,
		duration_ms INTEGER,
		started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		finished_at TIMESTAMP,
		source TEXT NOT NULL DEFAULT 'scheduled'
	);

	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS runner_images (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		image TEXT NOT NULL,
		description TEXT DEFAULT '',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS job_visibility (
		job_id          INTEGER NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
		target_type     TEXT NOT NULL CHECK (target_type IN ('user','role')),
		target_user_id  INTEGER REFERENCES users(id),
		target_role     TEXT
	);

	CREATE UNIQUE INDEX IF NOT EXISTS idx_job_visibility_unique
		ON job_visibility(job_id, target_type,
		                  COALESCE(target_user_id, -1),
		                  COALESCE(target_role, ''));
	`
	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	// migrate jobs table from old schema (image TEXT) to new schema (image_id FK)
	var hasImageCol int
	if err := db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('jobs') WHERE name = 'image'").Scan(&hasImageCol); err != nil {
		return fmt.Errorf("migrate check jobs schema: %w", err)
	}
	if hasImageCol > 0 {
		if _, err := db.Exec("DROP TABLE IF EXISTS jobs"); err != nil {
			return fmt.Errorf("migrate drop old jobs table: %w", err)
		}
		if _, err := db.Exec(schemaJobs); err != nil {
			return fmt.Errorf("migrate recreate jobs table: %w", err)
		}
	}

	var hasHostMappings int
	if err := db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('jobs') WHERE name = 'host_mappings'").Scan(&hasHostMappings); err != nil {
		return fmt.Errorf("migrate check jobs host_mappings: %w", err)
	}
	if hasHostMappings == 0 {
		if _, err := db.Exec("ALTER TABLE jobs ADD COLUMN host_mappings TEXT DEFAULT '[]'"); err != nil {
			return fmt.Errorf("migrate add jobs host_mappings: %w", err)
		}
	}

	var hasTokenFile int
	if err := db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('jobs') WHERE name = 'token_file'").Scan(&hasTokenFile); err != nil {
		return fmt.Errorf("migrate check jobs token_file: %w", err)
	}
	if hasTokenFile > 0 {
		if _, err := db.Exec("ALTER TABLE jobs DROP COLUMN token_file"); err != nil {
			return fmt.Errorf("migrate drop jobs token_file: %w", err)
		}
	}

	var hasSource int
	if err := db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('executions') WHERE name = 'source'").Scan(&hasSource); err != nil {
		return fmt.Errorf("migrate check executions source: %w", err)
	}
	if hasSource == 0 {
		if _, err := db.Exec("ALTER TABLE executions ADD COLUMN source TEXT NOT NULL DEFAULT 'scheduled'"); err != nil {
			return fmt.Errorf("migrate add executions source: %w", err)
		}
	}

	return nil
}
