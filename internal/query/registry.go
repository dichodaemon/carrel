package query

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cespare/xxhash/v2"
	"github.com/google/uuid"

	"github.com/dichodaemon/carrel/internal/composer"
	"github.com/dichodaemon/carrel/internal/registry"
)

// queryImpl implements QueryRegistry backed by an underlying registry.
type queryImpl struct {
	registry.Registry
}

// New creates a QueryRegistry wrapping a base Registry.
func New(reg registry.Registry) QueryRegistry {
	return &queryImpl{Registry: reg}
}

func (q *queryImpl) ListFiles(opts FileQueryOpts) ([]FileResult, error) {
	sources, err := q.ListSources()
	if err != nil {
		return nil, err
	}
	sourceMap := make(map[uuid.UUID]registry.Source)
	for _, s := range sources {
		sourceMap[s.ID] = s
	}

	var sourceIDs []uuid.UUID
	for _, s := range sources {
		sourceIDs = append(sourceIDs, s.ID)
	}
	entries, err := q.ResolveEntries(sourceIDs)
	if err != nil {
		return nil, err
	}

	var results []FileResult
	for _, e := range entries {
		if opts.Type != nil && e.Type != *opts.Type {
			continue
		}
		if opts.Source != nil {
			src, ok := sourceMap[e.SourceID]
			if !ok || src.Alias != *opts.Source {
				continue
			}
		}

		src := sourceMap[e.SourceID]
		fullPath := filepath.Join(src.Path, e.RelativePath)
		_, statErr := os.Stat(fullPath)
		status := FileExists
		if statErr != nil {
			status = FileMissing
		}

		if opts.Valid && status != FileExists {
			continue
		}
		if opts.Orphaned && status != FileMissing {
			continue
		}

		results = append(results, FileResult{
			SourceAlias: src.Alias,
			Type:        e.Type,
			TypeName:    typeName(e.Type),
			Name:        e.Name,
			Path:        fullPath,
			Status:      status,
			ContentHash: e.ContentHash,
			Final:       e.Final,
			ID:          fmt.Sprintf("%s:%s:%s", src.Alias, typeName(e.Type), e.Name),
		})
	}
	return results, nil
}

func (q *queryImpl) ListDeployed(opts DeployQueryOpts) ([]DeployResult, error) {
	consumers, err := q.ListConsumers()
	if err != nil {
		return nil, err
	}

	var results []DeployResult
	for _, c := range consumers {
		if opts.Consumer != nil && c.Alias != *opts.Consumer {
			continue
		}

		_, entries, err := q.LastDeployment(c.ID)
		if err == registry.ErrNoDeployment {
			continue
		}
		if err != nil {
			return nil, err
		}

		for _, e := range entries {
			var claimHash, actualHash int64
			var status DeployStatus

			if opts.Actual && !opts.Claims {
				// Filesystem-only mode
				data, fErr := os.ReadFile(e.Path)
				if fErr != nil {
					continue
				}
				actualHash = hashData(data)
				status = DeployForeign
				if e.ContentHash == actualHash {
					status = DeployDeployed
				}
			} else if opts.Claims && !opts.Actual {
				// Claims-only mode
				claimHash = e.ContentHash
				status = DeployDeployed
			} else {
				// Default: compare claim vs disk
				claimHash = e.ContentHash
				data, fErr := os.ReadFile(e.Path)
				if fErr != nil {
					status = DeployMissing
				} else {
					actualHash = hashData(data)
					if actualHash == claimHash {
						status = DeployDeployed
					} else {
						status = DeployModified
					}
				}
			}

			results = append(results, DeployResult{
				ConsumerAlias: c.Alias,
				Path:          e.Path,
				Status:        status,
				ClaimHash:     claimHash,
				ActualHash:    actualHash,
				ID:            fmt.Sprintf("%s:%s", c.Alias, e.Path),
			})
		}
	}
	return results, nil
}

func (q *queryImpl) Plan(consumerAlias string) ([]PlanResult, error) {
	c, err := q.ResolveConsumer(consumerAlias)
	if err != nil {
		return nil, fmt.Errorf("consumer %q: %w", consumerAlias, err)
	}

	rSlots, err := q.ResolveSlots(c.ID)
	if err != nil {
		return nil, fmt.Errorf("resolve slots: %w", err)
	}

	// Build composer slots and entry map
	var cSlots []composer.Slot
	entrySlots := make(map[uuid.UUID][]composer.Entry)
	entryLookup := make(map[uuid.UUID]registry.Entry)
	sourcePaths := make(map[uuid.UUID]string)

	for _, s := range rSlots {
		cSlots = append(cSlots, composer.Slot{
			ID:              s.ID,
			ConsumerID:      s.ConsumerID,
			Name:            s.Name,
			DestinationPath: c.DeployRoot + "/" + s.DestPath,
		})

		entries, eErr := q.ResolveEntrySlots(s.ID)
		if eErr != nil {
			continue
		}
		var cEntries []composer.Entry
		for _, se := range entries {
			cEntries = append(cEntries, composer.Entry{
				ID:       se.ID,
				SourceID: se.SourceID,
				Mode:     composer.ComposeMode(se.ComposeMode),
				Final:    se.Final,
				Priority: se.Priority,
			})
			entryLookup[se.ID] = se.Entry
		}
		entrySlots[s.ID] = cEntries
	}

	// Resolve source paths for content resolution
	sources, _ := q.ListSources()
	for _, s := range sources {
		sourcePaths[s.ID] = s.Path
	}

	plan, err := composer.Compose(c.ID, cSlots, entrySlots)
	if err != nil {
		return nil, fmt.Errorf("compose: %w", err)
	}

	// Resolve content
	for i := range plan.Files {
		resolvePlanContent(&plan.Files[i], entryLookup, sourcePaths)
	}

	// Compare against last deployment
	_, prevEntries, _ := q.LastDeployment(c.ID)
	prevMap := make(map[string]int64)
	for _, e := range prevEntries {
		prevMap[e.Path] = e.ContentHash
	}

	newMap := make(map[string]bool)
	var results []PlanResult
	for _, f := range plan.Files {
		newMap[f.DestinationPath] = true
		prevHash, existed := prevMap[f.DestinationPath]
		if !existed {
			results = append(results, PlanResult{
				ConsumerAlias: c.Alias,
				Path:          f.DestinationPath,
				Change:        PlanAdded,
				PlanHash:      f.ContentHash,
			})
		} else if prevHash != f.ContentHash {
			results = append(results, PlanResult{
				ConsumerAlias: c.Alias,
				Path:          f.DestinationPath,
				Change:        PlanModified,
				ClaimHash:     prevHash,
				PlanHash:      f.ContentHash,
			})
		}
	}

	// Removed files
	for path, hash := range prevMap {
		if !newMap[path] {
			results = append(results, PlanResult{
				ConsumerAlias: c.Alias,
				Path:          path,
				Change:        PlanRemoved,
				ClaimHash:     hash,
			})
		}
	}

	return results, nil
}

func (q *queryImpl) ListDependents(sourceAlias string) ([]ConsumerResult, error) {
	src, err := q.ResolveConsumer(sourceAlias)
	if err == nil {
		_ = src
	}
	// Resolve as source alias
	sources, err := q.ListSources()
	if err != nil {
		return nil, err
	}
	var sourceID uuid.UUID
	found := false
	for _, s := range sources {
		if s.Alias == sourceAlias {
			sourceID = s.ID
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("source %q not found", sourceAlias)
	}

	// Query consumer_sources join table
	consumers, err := q.ListConsumers()
	if err != nil {
		return nil, err
	}

	var results []ConsumerResult
	for _, c := range consumers {
		// Check if this consumer includes this source
		feeds, err := q.ResolveSources(c.ID)
		if err != nil {
			continue
		}
		for _, s := range feeds {
			if s.ID == sourceID {
				results = append(results, ConsumerResult{Alias: c.Alias, Path: c.Path})
				break
			}
		}
	}
	return results, nil
}

func (q *queryImpl) ListFeeds(consumerAlias string) ([]SourceResult, error) {
	c, err := q.ResolveConsumer(consumerAlias)
	if err != nil {
		return nil, fmt.Errorf("consumer %q: %w", consumerAlias, err)
	}

	sources, err := q.ResolveSources(c.ID)
	if err != nil {
		return nil, err
	}

	var results []SourceResult
	for _, s := range sources {
		results = append(results, SourceResult{
			Alias: s.Alias,
			Path:  s.Path,
			Scope: s.Scope,
		})
	}
	return results, nil
}

func (q *queryImpl) TraceSource(sourceAlias string, typ registry.CapabilityType, name string) ([]TraceResult, error) {
	sources, _ := q.ListSources()
	var sourceID uuid.UUID
	for _, s := range sources {
		if s.Alias == sourceAlias {
			sourceID = s.ID
			break
		}
	}

	var sourceIDs []uuid.UUID
	for _, s := range sources {
		sourceIDs = append(sourceIDs, s.ID)
	}
	entries, _ := q.ResolveEntries(sourceIDs)

	var entryID uuid.UUID
	for _, e := range entries {
		if e.SourceID == sourceID && e.Type == typ && e.Name == name {
			entryID = e.ID
			break
		}
	}
	if entryID == uuid.Nil {
		return nil, fmt.Errorf("entry %s:%s:%s not found", sourceAlias, typeName(typ), name)
	}

	// Query deployment_trace via consumers
	consumers, _ := q.ListConsumers()
	var results []TraceResult
	for _, c := range consumers {
		_, depEntries, err := q.LastDeployment(c.ID)
		if err != nil {
			continue
		}
		for _, de := range depEntries {
			if de.SourceEntry == entryID {
				results = append(results, TraceResult{
					ConsumerAlias: c.Alias,
					DeployedPath:  de.Path,
				})
			}
		}
	}
	return results, nil
}

func (q *queryImpl) TraceDeployed(consumerAlias string, path string) ([]TraceResult, error) {
	c, err := q.ResolveConsumer(consumerAlias)
	if err != nil {
		return nil, fmt.Errorf("consumer %q: %w", consumerAlias, err)
	}

	_, depEntries, err := q.LastDeployment(c.ID)
	if err != nil {
		return nil, err
	}

	var sourceEntryID uuid.UUID
	for _, de := range depEntries {
		if de.Path == path {
			sourceEntryID = de.SourceEntry
			break
		}
	}
	if sourceEntryID == uuid.Nil {
		return nil, fmt.Errorf("deployed file %s:%s not found", consumerAlias, path)
	}

	// Resolve source entry details
	sources, _ := q.ListSources()
	var sourceIDs []uuid.UUID
	for _, s := range sources {
		sourceIDs = append(sourceIDs, s.ID)
	}
	entries, _ := q.ResolveEntries(sourceIDs)

	sourceMap := make(map[uuid.UUID]registry.Source)
	for _, s := range sources {
		sourceMap[s.ID] = s
	}

	var results []TraceResult
	for _, e := range entries {
		if e.ID == sourceEntryID {
			src := sourceMap[e.SourceID]
			results = append(results, TraceResult{
				SourceAlias: src.Alias,
				SourceType:  e.Type,
				SourceName:  e.Name,
				SourcePath:  filepath.Join(src.Path, e.RelativePath),
			})
		}
	}
	return results, nil
}

func resolvePlanContent(f *composer.OutputFile, entryLookup map[uuid.UUID]registry.Entry, sourcePaths map[uuid.UUID]string) {
	var allContent []byte
	for _, eID := range f.SourceEntries {
		e, ok := entryLookup[eID]
		if !ok {
			continue
		}
		srcPath, ok := sourcePaths[e.SourceID]
		if !ok {
			continue
		}
		filePath := filepath.Join(srcPath, e.RelativePath)
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}
		allContent = append(allContent, data...)
	}
	f.Content = allContent
	f.ContentHash = hashData(allContent)
}

func (q *queryImpl) ListSlots(consumerAlias string) ([]registry.Slot, error) {
	c, err := q.ResolveConsumer(consumerAlias)
	if err != nil {
		return nil, fmt.Errorf("consumer %q: %w", consumerAlias, err)
	}
	return q.ResolveSlots(c.ID)
}

func (q *queryImpl) ListSlotEntries(slotID uuid.UUID) ([]registry.Entry, error) {
	se, err := q.ResolveEntrySlots(slotID)
	if err != nil {
		return nil, err
	}
	out := make([]registry.Entry, len(se))
	for i, s := range se {
		out[i] = s.Entry
	}
	return out, nil
}

func typeName(typ registry.CapabilityType) string {
	names := map[registry.CapabilityType]string{
		registry.TypeRule:          "rule",
		registry.TypeSkill:         "skill",
		registry.TypeCommand:       "command",
		registry.TypeExtension:     "extension",
		registry.TypeAgent:         "agent",
		registry.TypeTool:          "tool",
		registry.TypeHook:          "hook",
		registry.TypePrompt:        "prompt",
		registry.TypeInstruction:   "instruction",
		registry.TypeContextFile:   "context-file",
		registry.TypeAppendSystem:  "append-system",
		registry.TypeZshConfig:     "zsh",
		registry.TypeNvimConfig:    "nvim",
		registry.TypeWeztermConfig: "wezterm",
		registry.TypeP10kConfig:    "p10k",
	}
	if n, ok := names[typ]; ok {
		return n
	}
	return fmt.Sprintf("type-%d", typ)
}

func hashData(data []byte) int64 {
	return int64(xxhash.Sum64(data))
}
