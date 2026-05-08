package registry

import "fmt"

const SchemaVersion = 6

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
	3: {
		`UPDATE schema_version SET version = 3`,

		`CREATE TABLE slots (
			id VARCHAR(36) PRIMARY KEY,
			consumer_id VARCHAR(36) NOT NULL REFERENCES consumers(id),
			name VARCHAR(255) NOT NULL,
			dest_path VARCHAR(1024) NOT NULL,
			compose_mode INT,
			UNIQUE (consumer_id, name)
		)`,

		`CREATE TABLE entry_slots (
			entry_id VARCHAR(36) NOT NULL REFERENCES entries(id),
			slot_id VARCHAR(36) NOT NULL REFERENCES slots(id),
			PRIMARY KEY (entry_id, slot_id)
		)`,

		`ALTER TABLE consumers ADD COLUMN deploy_root TEXT`,
		`ALTER TABLE entries ADD COLUMN compose_mode INT`,
	},
	4: {
		`UPDATE schema_version SET version = 4`,
		`ALTER TABLE consumers DROP INDEX path`,
	},
	5: {
		`UPDATE schema_version SET version = 5`,
		`ALTER TABLE entry_slots ADD COLUMN priority INT NOT NULL DEFAULT 0`,
	},
	6: {
		`UPDATE schema_version SET version = 6`,
		`ALTER TABLE entries DROP COLUMN final`,
		`ALTER TABLE entries DROP COLUMN primitive_override`,
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

// BackfillV3 runs data migration for v3 schema.
// Must be called after ApplyMigrations when upgrading from v2.
func BackfillV3(r *DoltRegistry) error {
	// Backfill deploy_root for existing consumers
	if _, err := r.db.Exec(`UPDATE consumers SET deploy_root = CONCAT(path, '/.omp') WHERE deploy_root IS NULL`); err != nil {
		return fmt.Errorf("backfill deploy_root: %w", err)
	}

	// Backfill compose_mode for entries where it is NULL.
	// Uses primitive_override if set, otherwise convention default.
	rows, err := r.db.Query(`SELECT e.id, e.type, e.primitive_override FROM entries e WHERE e.compose_mode IS NULL`)
	if err != nil {
		return fmt.Errorf("backfill query entries: %w", err)
	}
	defer rows.Close()

	type toUpdate struct {
		id   string
		mode int
	}
	var updates []toUpdate
	for rows.Next() {
		var id string
		var typ int
		var primOverride *int
		if err := rows.Scan(&id, &typ, &primOverride); err != nil {
			return fmt.Errorf("backfill scan: %w", err)
		}
		var mode int
		if primOverride != nil {
			mode = *primOverride
		} else {
			conv, ok := Conventions[CapabilityType(typ)]
			if ok {
				mode = int(conv.DefaultPrimitive)
			}
		}
		updates = append(updates, toUpdate{id, mode})
	}

	for _, u := range updates {
		if _, err := r.db.Exec(`UPDATE entries SET compose_mode = ? WHERE id = ?`, u.mode, u.id); err != nil {
			return fmt.Errorf("backfill update %s: %w", u.id, err)
		}
	}
	return nil
}
