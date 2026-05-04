package registry

// SchemaVersion is the current registry schema version.
// Increment when schema changes; migrations are applied in order.
const SchemaVersion = 1

// Migrations returns SQL statements to initialize the registry schema.
// Each entry is a versioned migration. v0 → v1 is the initial schema.
var Migrations = map[int][]string{
	1: {
		`CREATE TABLE schema_version (version INT PRIMARY KEY)`,
		`INSERT INTO schema_version (version) VALUES (1)`,

		`CREATE TABLE consumers (
			id VARCHAR(36) PRIMARY KEY,
			alias VARCHAR(255) NOT NULL UNIQUE,
			path VARCHAR(1024) NOT NULL UNIQUE,
			kind INT NOT NULL DEFAULT 0
		)`,

		`CREATE TABLE sources (
			id VARCHAR(36) PRIMARY KEY,
			alias VARCHAR(255) NOT NULL UNIQUE,
			path VARCHAR(1024) NOT NULL,
			scope INT NOT NULL DEFAULT 0,
			kind INT NOT NULL DEFAULT 0
		)`,

		`CREATE TABLE entries (
			id VARCHAR(36) PRIMARY KEY,
			source_id VARCHAR(36) NOT NULL REFERENCES sources(id),
			name VARCHAR(255) NOT NULL,
			type INT NOT NULL DEFAULT 0,
			relative_path VARCHAR(1024) NOT NULL,
			content_hash BIGINT NOT NULL DEFAULT 0,
			final BOOLEAN NOT NULL DEFAULT FALSE,
			primitive_override INT,
			created_by INT NOT NULL DEFAULT 0
		)`,

		`CREATE TABLE deployments (
			id VARCHAR(36) PRIMARY KEY,
			consumer_id VARCHAR(36) NOT NULL REFERENCES consumers(id),
			attempted_at TIMESTAMP NOT NULL,
			succeeded_at TIMESTAMP,
			config_hash BIGINT NOT NULL DEFAULT 0
		)`,

		`CREATE TABLE deployment_entries (
			deployment_id VARCHAR(36) NOT NULL REFERENCES deployments(id),
			path VARCHAR(1024) NOT NULL,
			content_hash BIGINT NOT NULL DEFAULT 0,
			source_entry VARCHAR(36) NOT NULL REFERENCES entries(id)
		)`,
	},
}

// ApplyMigrations ensures the schema is at the current version.
func ApplyMigrations(r *DoltRegistry) error {
	// Create database if not exists (embedded Dolt)
	_, err := r.db.Exec("CREATE DATABASE IF NOT EXISTS registry")
	if err != nil {
		return err
	}
	_, err = r.db.Exec("USE registry")
	if err != nil {
		return err
	}

	// Check current version
	var currentVersion int
	row := r.db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_version")
	if err := row.Scan(&currentVersion); err != nil {
		// schema_version table doesn't exist yet — start from 0
		currentVersion = 0
	}

	// Apply migrations in order
	for v := currentVersion + 1; v <= SchemaVersion; v++ {
		stmts, ok := Migrations[v]
		if !ok {
			continue
		}
		for _, stmt := range stmts {
			if _, err := r.db.Exec(stmt); err != nil {
				return err
			}
		}
	}

	return nil
}
