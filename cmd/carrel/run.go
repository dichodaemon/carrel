package main

import (
	"os"
	"path/filepath"

	"github.com/cespare/xxhash/v2"
	"github.com/google/uuid"

	"github.com/dichodaemon/carrel/internal/composer"
	"github.com/dichodaemon/carrel/internal/registry"
)

// resolveOutputContent populates the Content and ContentHash fields of an
// OutputFile by reading the contributing source files from disk.
func resolveOutputContent(f *composer.OutputFile, entryLookup map[uuid.UUID]registry.Entry, sourceByID map[uuid.UUID]registry.Source) {
	var allContent []byte
	for _, eID := range f.SourceEntries {
		e, ok := entryLookup[eID]
		if !ok {
			continue
		}
		src, ok := sourceByID[e.SourceID]
		if !ok {
			continue
		}
		filePath := filepath.Join(src.Path, e.RelativePath)
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}
		allContent = append(allContent, data...)
	}

	f.Content = allContent
	f.ContentHash = int64(xxhash.Sum64(allContent))
}
