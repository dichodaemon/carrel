package authoring

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cespare/xxhash/v2"
	"github.com/google/uuid"

	"github.com/dichodaemon/carrel/internal/registry"
)

// AddEntry creates a file at the conventional path within the named source
// and registers the entry in the registry.
func AddEntry(reg registry.Registry, typ registry.CapabilityType, name string, sourceAlias string, content []byte) (registry.Entry, error) {
	relPath, err := registry.RelativePath(typ, name)
	if err != nil {
		return registry.Entry{}, fmt.Errorf("relative path: %w", err)
	}

	source, err := resolveSource(reg, sourceAlias)
	if err != nil {
		return registry.Entry{}, err
	}

	fullPath := filepath.Join(source.Path, relPath)
	if err := writeFile(fullPath, content); err != nil {
		return registry.Entry{}, err
	}

	entry := registry.Entry{
		ID:           uuid.New(),
		SourceID:     source.ID,
		Name:         name,
		Type:         typ,
		RelativePath: relPath,
		ContentHash:  int64(xxhash.Sum64(content)),
		CreatedBy:    registry.OriginCarrel,
	}

	if err := reg.RegisterEntry(entry); err != nil {
		return registry.Entry{}, fmt.Errorf("register entry: %w", err)
	}

	return entry, nil
}

// RemoveEntry removes the file and registry entry matching the given type and name.
func RemoveEntry(reg registry.Registry, typ registry.CapabilityType, name string) error {
	entry, source, err := findEntry(reg, typ, name)
	if err != nil {
		return err
	}

	fullPath := filepath.Join(source.Path, entry.RelativePath)
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove %s: %w", fullPath, err)
	}

	if err := reg.RemoveEntry(entry.ID); err != nil {
		return fmt.Errorf("remove entry from registry: %w", err)
	}

	return nil
}

// EditEntry overwrites the file content and updates the ContentHash in the registry.
func EditEntry(reg registry.Registry, typ registry.CapabilityType, name string, newContent []byte) error {
	entry, source, err := findEntry(reg, typ, name)
	if err != nil {
		return err
	}

	fullPath := filepath.Join(source.Path, entry.RelativePath)
	if err := writeFile(fullPath, newContent); err != nil {
		return err
	}

	newHash := int64(xxhash.Sum64(newContent))
	entry.ContentHash = newHash

	// Re-register to update the stored entry.
	if err := reg.RegisterEntry(entry); err != nil {
		return fmt.Errorf("re-register entry: %w", err)
	}

	return nil
}

// UpdateEntryMeta updates metadata fields (Final, PrimitiveOverride) on an entry.
func UpdateEntryMeta(reg registry.Registry, typ registry.CapabilityType, name string, updates registry.MetaUpdates) error {
	entry, _, err := findEntry(reg, typ, name)
	if err != nil {
		return err
	}

	if err := reg.UpdateEntryMeta(entry.ID, updates); err != nil {
		return fmt.Errorf("update entry meta: %w", err)
	}

	return nil
}

// RenameEntry moves the file to the new name's conventional path and updates
// the registry, preserving the entry's UUID and other fields.
func RenameEntry(reg registry.Registry, typ registry.CapabilityType, oldName string, newName string) error {
	entry, source, err := findEntry(reg, typ, oldName)
	if err != nil {
		return err
	}

	newRelPath, err := registry.RelativePath(typ, newName)
	if err != nil {
		return fmt.Errorf("relative path for %q: %w", newName, err)
	}

	oldFullPath := filepath.Join(source.Path, entry.RelativePath)
	newFullPath := filepath.Join(source.Path, newRelPath)

	// Ensure the target directory exists.
	if dir := filepath.Dir(newFullPath); dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}

	if err := os.Rename(oldFullPath, newFullPath); err != nil {
		return fmt.Errorf("rename %s -> %s: %w", oldFullPath, newFullPath, err)
	}

	// Update the entry in place by re-registering with the same UUID.
	entry.Name = newName
	entry.RelativePath = newRelPath
	if err := reg.RegisterEntry(entry); err != nil {
		return fmt.Errorf("re-register entry: %w", err)
	}

	return nil
}

// resolveSource returns the source matching alias.
func resolveSource(reg registry.Registry, alias string) (registry.Source, error) {
	sources, err := reg.ListSources()
	if err != nil {
		return registry.Source{}, fmt.Errorf("list sources: %w", err)
	}
	for _, s := range sources {
		if s.Alias == alias {
			return s, nil
		}
	}
	return registry.Source{}, fmt.Errorf("source %q: %w", alias, registry.ErrNotFound)
}

// findEntry locates an entry by type and name across all sources.
// Returns ErrNotFound if no match, or an error if multiple entries match.
func findEntry(reg registry.Registry, typ registry.CapabilityType, name string) (registry.Entry, registry.Source, error) {
	sources, err := reg.ListSources()
	if err != nil {
		return registry.Entry{}, registry.Source{}, fmt.Errorf("list sources: %w", err)
	}

	var sourceIDs []uuid.UUID
	sourceByID := make(map[uuid.UUID]registry.Source)
	for _, s := range sources {
		sourceIDs = append(sourceIDs, s.ID)
		sourceByID[s.ID] = s
	}

	entries, err := reg.ResolveEntries(sourceIDs)
	if err != nil {
		return registry.Entry{}, registry.Source{}, fmt.Errorf("resolve entries: %w", err)
	}

	var match *registry.Entry
	for i, e := range entries {
		if e.Type == typ && e.Name == name {
			if match != nil {
				return registry.Entry{}, registry.Source{}, fmt.Errorf("multiple entries found for type=%d name=%q", typ, name)
			}
			match = &entries[i]
		}
	}

	if match == nil {
		return registry.Entry{}, registry.Source{}, fmt.Errorf("entry type=%d name=%q: %w", typ, name, registry.ErrNotFound)
	}

	source, ok := sourceByID[match.SourceID]
	if !ok {
		return registry.Entry{}, registry.Source{}, fmt.Errorf("source %s for entry: %w", match.SourceID, registry.ErrNotFound)
	}

	return *match, source, nil
}

// writeFile creates parent directories and writes content to path.
func writeFile(path string, content []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	if err := os.WriteFile(path, content, 0644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// Ensure errors.Is works with registry error types.
var _ = errors.Is
