package registry

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	_ "github.com/dolthub/driver"
)

// DoltRegistry is a Dolt-backed Registry implementation.
type DoltRegistry struct {
	db *sql.DB
}

// NewDoltRegistry opens a Dolt-backed registry at the given data directory.
// The directory must exist before calling.
func NewDoltRegistry(dsn string) (*DoltRegistry, error) {
	db, err := sql.Open("dolt", dsn)
	if err != nil {
		return nil, fmt.Errorf("registry: open: %w", err)
	}
	return &DoltRegistry{db: db}, nil
}


// DB returns the underlying database handle (for migrations and testing).
func (r *DoltRegistry) DB() *sql.DB {
	return r.db
}

// Close closes the underlying database connection.
func (r *DoltRegistry) Close() error {
	return r.db.Close()
}

// RegisterConsumer implements Registry.
func (r *DoltRegistry) RegisterConsumer(c Consumer) error {
	_, err := r.db.Exec(
		`INSERT INTO consumers (id, alias, path, kind) VALUES (?, ?, ?, ?)`,
		c.ID.String(), c.Alias, c.Path, int(c.Kind),
	)
	return err
}

// ResolveConsumer implements Registry.
func (r *DoltRegistry) ResolveConsumer(aliasOrPath string) (Consumer, error) {
	row := r.db.QueryRow(
		`SELECT id, alias, path, kind FROM consumers WHERE alias = ? OR path = ?`,
		aliasOrPath, aliasOrPath,
	)
	var c Consumer
	var idStr string
	var kind int
	if err := row.Scan(&idStr, &c.Alias, &c.Path, &kind); err != nil {
		if err == sql.ErrNoRows {
			return Consumer{}, ErrNotFound
		}
		return Consumer{}, err
	}
	c.ID, _ = parseUUID(idStr)
	c.Kind = ConsumerKind(kind)
	return c, nil
}

// ListConsumers implements Registry.
func (r *DoltRegistry) ListConsumers() ([]Consumer, error) {
	rows, err := r.db.Query(`SELECT id, alias, path, kind FROM consumers ORDER BY alias`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanConsumers(rows)
}

// RegisterSource implements Registry.
func (r *DoltRegistry) RegisterSource(s Source) error {
	_, err := r.db.Exec(
		`INSERT INTO sources (id, alias, path, scope, kind) VALUES (?, ?, ?, ?, ?)`,
		s.ID.String(), s.Alias, s.Path, int(s.Scope), int(s.Kind),
	)
	return err
}

// ResolveSources implements Registry.
func (r *DoltRegistry) ResolveSources(consumerID uuid.UUID) ([]Source, error) {
	// Return universal sources first, then target-specific for this consumer.
	rows, err := r.db.Query(
		`SELECT s.id, s.alias, s.path, s.scope, s.kind FROM sources s
		 WHERE s.scope = ? OR s.scope = ?
		 ORDER BY s.scope ASC`,
		int(ScopeUniversal), int(ScopeTargetSpecific),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSources(rows)
}

// ListSources implements Registry.
func (r *DoltRegistry) ListSources() ([]Source, error) {
	rows, err := r.db.Query(`SELECT id, alias, path, scope, kind FROM sources ORDER BY scope, alias`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSources(rows)
}

// RegisterEntry implements Registry.
func (r *DoltRegistry) RegisterEntry(e Entry) error {
	var primOverride *int
	if e.PrimitiveOverride != nil {
		v := int(*e.PrimitiveOverride)
		primOverride = &v
	}
	_, err := r.db.Exec(
		`INSERT INTO entries (id, source_id, name, type, relative_path, content_hash, final, primitive_override, created_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID.String(), e.SourceID.String(), e.Name, int(e.Type),
		e.RelativePath, e.ContentHash, e.Final, primOverride, int(e.CreatedBy),
	)
	return err
}

// ResolveEntries implements Registry.
func (r *DoltRegistry) ResolveEntries(sourceIDs []uuid.UUID) ([]Entry, error) {
	if len(sourceIDs) == 0 {
		return nil, nil
	}
	query := `SELECT id, source_id, name, type, relative_path, content_hash, final, primitive_override, created_by FROM entries WHERE source_id IN (`
	args := make([]interface{}, len(sourceIDs))
	for i, id := range sourceIDs {
		if i > 0 {
			query += ","
		}
		query += "?"
		args[i] = id.String()
	}
	query += `) ORDER BY type, name`
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEntries(rows)
}

// RemoveEntry implements Registry.
func (r *DoltRegistry) RemoveEntry(entryID uuid.UUID) error {
	result, err := r.db.Exec(`DELETE FROM entries WHERE id = ?`, entryID.String())
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateEntryMeta implements Registry.
func (r *DoltRegistry) UpdateEntryMeta(entryID uuid.UUID, updates MetaUpdates) error {
	if updates.Final == nil && updates.PrimitiveOverride == nil {
		return nil // nothing to update
	}
	setClauses := []string{}
	args := []interface{}{}

	if updates.Final != nil {
		setClauses = append(setClauses, "final = ?")
		args = append(args, *updates.Final)
	}
	if updates.PrimitiveOverride != nil {
		if *updates.PrimitiveOverride == nil {
			setClauses = append(setClauses, "primitive_override = NULL")
		} else {
			setClauses = append(setClauses, "primitive_override = ?")
			args = append(args, int(**updates.PrimitiveOverride))
		}
	}

	query := "UPDATE entries SET "
	for i, clause := range setClauses {
		if i > 0 {
			query += ", "
		}
		query += clause
	}
	query += " WHERE id = ?"
	args = append(args, entryID.String())

	result, err := r.db.Exec(query, args...)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// RecordDeployment implements Registry.
func (r *DoltRegistry) RecordDeployment(d Deployment, entries []DeploymentEntry) error {
	_, err := r.db.Exec(
		`INSERT INTO deployments (id, consumer_id, attempted_at, succeeded_at, config_hash) VALUES (?, ?, ?, ?, ?)`,
		d.ID.String(), d.ConsumerID.String(), d.AttemptedAt, d.SucceededAt, d.ConfigHash,
	)
	if err != nil {
		return err
	}
	for _, e := range entries {
		_, err := r.db.Exec(
			`INSERT INTO deployment_entries (deployment_id, path, content_hash, source_entry) VALUES (?, ?, ?, ?)`,
			d.ID.String(), e.Path, e.ContentHash, e.SourceEntry.String(),
		)
		if err != nil {
			return err
		}
	}
	return nil
}

// LastDeployment implements Registry.
func (r *DoltRegistry) LastDeployment(consumerID uuid.UUID) (Deployment, []DeploymentEntry, error) {
	row := r.db.QueryRow(
		`SELECT id, consumer_id, attempted_at, succeeded_at, config_hash FROM deployments
		 WHERE consumer_id = ? AND succeeded_at IS NOT NULL
		 ORDER BY attempted_at DESC LIMIT 1`,
		consumerID.String(),
	)
	var d Deployment
	var idStr, consIDStr string
	var succeededAt sql.NullTime
	if err := row.Scan(&idStr, &consIDStr, &d.AttemptedAt, &succeededAt, &d.ConfigHash); err != nil {
		if err == sql.ErrNoRows {
			return Deployment{}, nil, ErrNoDeployment
		}
		return Deployment{}, nil, err
	}
	d.ID, _ = parseUUID(idStr)
	d.ConsumerID, _ = parseUUID(consIDStr)
	if succeededAt.Valid {
		d.SucceededAt = &succeededAt.Time
	}

	rows, err := r.db.Query(
		`SELECT deployment_id, path, content_hash, source_entry FROM deployment_entries WHERE deployment_id = ?`,
		idStr,
	)
	if err != nil {
		return Deployment{}, nil, err
	}
	defer rows.Close()

	var entries []DeploymentEntry
	for rows.Next() {
		var e DeploymentEntry
		var depIDStr, srcEntryStr string
		if err := rows.Scan(&depIDStr, &e.Path, &e.ContentHash, &srcEntryStr); err != nil {
			return Deployment{}, nil, err
		}
		e.DeploymentID, _ = parseUUID(depIDStr)
		e.SourceEntry, _ = parseUUID(srcEntryStr)
		entries = append(entries, e)
	}
	return d, entries, nil
}

// scanConsumers reads consumer rows.
func scanConsumers(rows *sql.Rows) ([]Consumer, error) {
	var out []Consumer
	for rows.Next() {
		var c Consumer
		var idStr string
		var kind int
		if err := rows.Scan(&idStr, &c.Alias, &c.Path, &kind); err != nil {
			return nil, err
		}
		c.ID, _ = parseUUID(idStr)
		c.Kind = ConsumerKind(kind)
		out = append(out, c)
	}
	return out, rows.Err()
}

// scanSources reads source rows.
func scanSources(rows *sql.Rows) ([]Source, error) {
	var out []Source
	for rows.Next() {
		var s Source
		var idStr string
		var scope, kind int
		if err := rows.Scan(&idStr, &s.Alias, &s.Path, &scope, &kind); err != nil {
			return nil, err
		}
		s.ID, _ = parseUUID(idStr)
		s.Scope = SourceScope(scope)
		s.Kind = SourceKind(kind)
		out = append(out, s)
	}
	return out, rows.Err()
}

// scanEntries reads entry rows.
func scanEntries(rows *sql.Rows) ([]Entry, error) {
	var out []Entry
	for rows.Next() {
		var e Entry
		var idStr, srcIDStr string
		var typ, createdBy int
		var primOverride *int
		if err := rows.Scan(&idStr, &srcIDStr, &e.Name, &typ, &e.RelativePath, &e.ContentHash, &e.Final, &primOverride, &createdBy); err != nil {
			return nil, err
		}
		e.ID, _ = parseUUID(idStr)
		e.SourceID, _ = parseUUID(srcIDStr)
		e.Type = CapabilityType(typ)
		e.CreatedBy = EntryOrigin(createdBy)
		if primOverride != nil {
			v := Primitive(*primOverride)
			e.PrimitiveOverride = &v
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// parseUUID parses a UUID string, returning uuid.Nil on failure.
func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}
