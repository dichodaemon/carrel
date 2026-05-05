package registry

const SchemaVersion = 2

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
	2: {
		`UPDATE schema_version SET version = 2`,

		`CREATE TABLE consumer_sources (
			consumer_id VARCHAR(36) NOT NULL REFERENCES consumers(id),
			source_id   VARCHAR(36) NOT NULL REFERENCES sources(id),
			PRIMARY KEY (consumer_id, source_id)
		)`,

		`CREATE TABLE deployment_trace (
			source_entry VARCHAR(36) NOT NULL REFERENCES entries(id),
			consumer_id  VARCHAR(36) NOT NULL REFERENCES consumers(id),
			path         VARCHAR(1024) NOT NULL,
			PRIMARY KEY (source_entry, consumer_id, path)
		)`,
	},
}

func ApplyMigrations(r *DoltRegistry) error {
	_, err := r.db.Exec("CREATE DATABASE IF NOT EXISTS registry")
	if err != nil {
		return err
	}
	_, err = r.db.Exec("USE registry")
	if err != nil {
		return err
	}

	var currentVersion int
	row := r.db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_version")
	if err := row.Scan(&currentVersion); err != nil {
		currentVersion = 0
	}

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
