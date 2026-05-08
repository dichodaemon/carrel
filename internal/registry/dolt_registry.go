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

// New creates a Registry backed by Dolt at the given path.
func New(path string) (Registry, error) {
	dsn := fmt.Sprintf("file://%s?commitname=Carrel&commitemail=carrel@localhost&database=registry", path)
	db, err := sql.Open("dolt", dsn)
	if err != nil {
		return nil, fmt.Errorf("registry: open: %w", err)
	}
	r := &DoltRegistry{db: db}
	if err := ApplyMigrations(r); err != nil {
		r.Close()
		return nil, fmt.Errorf("registry: apply migrations: %w", err)
	}
	if err := BackfillV3(r); err != nil {
		r.Close()
		return nil, fmt.Errorf("registry: backfill v3: %w", err)
	}
	return r, nil
}

// NewMem creates an in-memory Registry for testing.
func NewMem() Registry { return NewMemRegistry() }

// NewDoltRegistry opens a Dolt-backed registry without migrations.
func NewDoltRegistry(dsn string) (*DoltRegistry, error) {
	db, err := sql.Open("dolt", dsn)
	if err != nil {
		return nil, fmt.Errorf("registry: open: %w", err)
	}
	return &DoltRegistry{db: db}, nil
}

// DB returns the underlying database handle (for tests and migrations).
func (r *DoltRegistry) DB() *sql.DB { return r.db }

// Close closes the connection.
func (r *DoltRegistry) Close() error { return r.db.Close() }

// RegisterConsumer implements Registry.
func (r *DoltRegistry) RegisterConsumer(c Consumer) error {
	_, err := r.db.Exec(
		`INSERT INTO consumers (id, alias, path, deploy_root, kind) VALUES (?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE
		   alias = VALUES(alias),
		   path = VALUES(path),
		   deploy_root = VALUES(deploy_root),
		   kind = VALUES(kind)`,
		c.ID.String(), c.Alias, c.Path, c.DeployRoot, int(c.Kind),
	)
	return err
}

// ResolveConsumer implements Registry.
func (r *DoltRegistry) ResolveConsumer(aliasOrPath string) (Consumer, error) {
	row := r.db.QueryRow(
		`SELECT id, alias, path, deploy_root, kind FROM consumers WHERE alias = ? OR path = ?`,
		aliasOrPath, aliasOrPath,
	)
	var c Consumer
	var idStr string
	var kind int
	if err := row.Scan(&idStr, &c.Alias, &c.Path, &c.DeployRoot, &kind); err != nil {
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
	rows, err := r.db.Query(`SELECT id, alias, path, deploy_root, kind FROM consumers ORDER BY alias`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanConsumers(rows)
}

// RegisterSource implements Registry.
func (r *DoltRegistry) RegisterSource(s Source) error {
	_, err := r.db.Exec(
		`INSERT INTO sources (id, alias, path, scope, kind) VALUES (?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE
		   alias = VALUES(alias),
		   path = VALUES(path),
		   scope = VALUES(scope),
		   kind = VALUES(kind)`,
		s.ID.String(), s.Alias, s.Path, int(s.Scope), int(s.Kind),
	)
	return err
}

// LinkConsumerSource implements Registry.
func (r *DoltRegistry) LinkConsumerSource(consumerID, sourceID uuid.UUID) error {
	_, err := r.db.Exec(
		`INSERT IGNORE INTO consumer_sources (consumer_id, source_id) VALUES (?, ?)`,
		consumerID.String(), sourceID.String(),
	)
	return err
}

// ResolveSources implements Registry.
func (r *DoltRegistry) ResolveSources(consumerID uuid.UUID) ([]Source, error) {
	rows, err := r.db.Query(
		`SELECT s.id, s.alias, s.path, s.scope, s.kind FROM sources s
		 WHERE s.scope = ? OR (s.scope = ? AND s.id IN (SELECT source_id FROM consumer_sources WHERE consumer_id = ?))
		 ORDER BY s.scope ASC`,
		int(ScopeUniversal), int(ScopeTargetSpecific), consumerID.String(),
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
	_, err := r.db.Exec(
		`INSERT INTO entries (id, source_id, name, type, relative_path, content_hash, compose_mode, created_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE
		   name = VALUES(name),
		   type = VALUES(type),
		   relative_path = VALUES(relative_path),
		   content_hash = VALUES(content_hash),
		   compose_mode = VALUES(compose_mode),
		   created_by = VALUES(created_by)`,
		e.ID.String(), e.SourceID.String(), e.Name, int(e.Type),
		e.RelativePath, e.ContentHash, int(e.ComposeMode), int(e.CreatedBy))
	return err
}

// ResolveEntries implements Registry.
func (r *DoltRegistry) ResolveEntries(sourceIDs []uuid.UUID) ([]Entry, error) {
	if len(sourceIDs) == 0 {
		return nil, nil
	}
	query := `SELECT id, source_id, name, type, relative_path, content_hash, compose_mode, created_by FROM entries WHERE source_id IN (`
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
	if updates.ComposeMode == nil {
		return nil
	}
	setClauses := []string{}
	args := []interface{}{}
	if updates.ComposeMode != nil {
		setClauses = append(setClauses, "compose_mode = ?")
		args = append(args, int(**updates.ComposeMode))
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

		// Also record in deployment_trace for provenance queries
		_, _ = r.db.Exec(
			`INSERT INTO deployment_trace (source_entry, consumer_id, path) VALUES (?, ?, ?)`,
			e.SourceEntry.String(), d.ConsumerID.String(), e.Path,
		)
	}
	return nil
}

// LastDeployment implements Registry.
func (r *DoltRegistry) LastDeployment(consumerID uuid.UUID) (Deployment, []DeploymentEntry, error) {
	row := r.db.QueryRow(
		`SELECT id, consumer_id, attempted_at, succeeded_at, config_hash FROM deployments
		 WHERE consumer_id = ? AND succeeded_at IS NOT NULL ORDER BY attempted_at DESC LIMIT 1`,
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
		`SELECT deployment_id, path, content_hash, source_entry FROM deployment_entries WHERE deployment_id = ?`, idStr,
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

// RegisterSlot implements Registry.
func (r *DoltRegistry) RegisterSlot(s Slot) error {
	var mode *int
	if s.ComposeMode != nil {
		v := int(*s.ComposeMode)
		mode = &v
	}
	_, err := r.db.Exec(
		`INSERT INTO slots (id, consumer_id, name, dest_path, compose_mode) VALUES (?, ?, ?, ?, ?)`,
		s.ID.String(), s.ConsumerID.String(), s.Name, s.DestPath, mode,
	)
	return err
}

// UpdateSlot implements Registry.
func (r *DoltRegistry) UpdateSlot(slotID uuid.UUID, updates SlotUpdates) error {
	if updates.Name == nil && updates.DestPath == nil && updates.ComposeMode == nil {
		return nil
	}
	setClauses := []string{}
	args := []interface{}{}
	if updates.Name != nil {
		setClauses = append(setClauses, "name = ?")
		args = append(args, *updates.Name)
	}
	if updates.DestPath != nil {
		setClauses = append(setClauses, "dest_path = ?")
		args = append(args, *updates.DestPath)
	}
	if updates.ComposeMode != nil {
		if *updates.ComposeMode == nil {
			setClauses = append(setClauses, "compose_mode = NULL")
		} else {
			setClauses = append(setClauses, "compose_mode = ?")
			args = append(args, int(**updates.ComposeMode))
		}
	}
	query := "UPDATE slots SET "
	for i, clause := range setClauses {
		if i > 0 {
			query += ", "
		}
		query += clause
	}
	query += " WHERE id = ?"
	args = append(args, slotID.String())
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

// RemoveSlot implements Registry.
func (r *DoltRegistry) RemoveSlot(slotID uuid.UUID) error {
	r.db.Exec(`DELETE FROM entry_slots WHERE slot_id = ?`, slotID.String())
	result, err := r.db.Exec(`DELETE FROM slots WHERE id = ?`, slotID.String())
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ResolveSlots implements Registry.
func (r *DoltRegistry) ResolveSlots(consumerID uuid.UUID) ([]Slot, error) {
	rows, err := r.db.Query(
		`SELECT id, consumer_id, name, dest_path, compose_mode FROM slots WHERE consumer_id = ? ORDER BY name`,
		consumerID.String(),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSlots(rows)
}

// LinkEntrySlot implements Registry.
func (r *DoltRegistry) LinkEntrySlot(entryID, slotID uuid.UUID, priority int) error {
	_, err := r.db.Exec(
		`INSERT INTO entry_slots (entry_id, slot_id, priority) VALUES (?, ?, ?)`,
		entryID.String(), slotID.String(), priority,
	)
	return err
}

// UnlinkEntrySlot implements Registry.
func (r *DoltRegistry) UnlinkEntrySlot(entryID, slotID uuid.UUID) error {
	result, err := r.db.Exec(
		`DELETE FROM entry_slots WHERE entry_id = ? AND slot_id = ?`,
		entryID.String(), slotID.String(),
	)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ResolveEntrySlots implements Registry.
func (r *DoltRegistry) ResolveEntrySlots(slotID uuid.UUID) ([]SlotEntry, error) {
	rows, err := r.db.Query(
		`SELECT e.id, e.source_id, e.name, e.type, e.relative_path, e.content_hash, e.compose_mode, e.created_by, es.priority
		 FROM entries e JOIN entry_slots es ON e.id = es.entry_id
		 WHERE es.slot_id = ? ORDER BY es.priority ASC, e.type, e.name`,
		slotID.String(),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSlotEntries(rows)
}

func scanConsumers(rows *sql.Rows) ([]Consumer, error) {
	var out []Consumer
	for rows.Next() {
		var c Consumer
		var idStr string
		var kind int
		if err := rows.Scan(&idStr, &c.Alias, &c.Path, &c.DeployRoot, &kind); err != nil {
			return nil, err
		}
		c.ID, _ = parseUUID(idStr)
		c.Kind = ConsumerKind(kind)
		out = append(out, c)
	}
	return out, rows.Err()
}

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

func scanEntries(rows *sql.Rows) ([]Entry, error) {
	var out []Entry
	for rows.Next() {
		var e Entry
		var idStr, srcIDStr string
		var typ, createdBy int
		var composeMode int
		if err := rows.Scan(&idStr, &srcIDStr, &e.Name, &typ, &e.RelativePath, &e.ContentHash, &composeMode, &createdBy); err != nil {
			return nil, err
		}
		e.ID, _ = parseUUID(idStr)
		e.SourceID, _ = parseUUID(srcIDStr)
		e.Type = CapabilityType(typ)
		e.CreatedBy = EntryOrigin(createdBy)
		e.ComposeMode = ComposeMode(composeMode)
		out = append(out, e)
	}
	return out, rows.Err()
}

func scanSlots(rows *sql.Rows) ([]Slot, error) {
	var out []Slot
	for rows.Next() {
		var s Slot
		var idStr, consIDStr string
		var mode *int
		if err := rows.Scan(&idStr, &consIDStr, &s.Name, &s.DestPath, &mode); err != nil {
			return nil, err
		}
		s.ID, _ = parseUUID(idStr)
		s.ConsumerID, _ = parseUUID(consIDStr)
		if mode != nil {
			v := ComposeMode(*mode)
			s.ComposeMode = &v
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func scanSlotEntries(rows *sql.Rows) ([]SlotEntry, error) {
	var out []SlotEntry
	for rows.Next() {
		var se SlotEntry
		var idStr, srcIDStr string
		var typ, createdBy int
		var composeMode int
		if err := rows.Scan(&idStr, &srcIDStr, &se.Name, &typ, &se.RelativePath, &se.ContentHash, &composeMode, &createdBy, &se.Priority); err != nil {
			return nil, err
		}
		se.ID, _ = parseUUID(idStr)
		se.SourceID, _ = parseUUID(srcIDStr)
		se.Type = CapabilityType(typ)
		se.ComposeMode = ComposeMode(composeMode)
		se.CreatedBy = EntryOrigin(createdBy)
		out = append(out, se)
	}
	return out, rows.Err()
}
func parseUUID(s string) (uuid.UUID, error) { return uuid.Parse(s) }
