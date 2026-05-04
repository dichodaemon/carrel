package composer

import (
	"github.com/google/uuid"

	"github.com/dichodaemon/carrel/internal/registry"
)

// OutputPlan is the result of composition.
type OutputPlan struct {
	Consumer  uuid.UUID
	Files     []OutputFile
	Overrides []OverrideRecord
}

// OutputFile is a single file to deploy.
type OutputFile struct {
	DestinationPath string
	Content         []byte
	ContentHash     uint64
	SourceEntries   []uuid.UUID // Which entries contributed
	Primitive       registry.Primitive
}

// OverrideRecord logs a collision resolved in favor of one entry.
type OverrideRecord struct {
	Path     string    // The slot where override occurred
	Winner   uuid.UUID // Entry that won
	Loser    uuid.UUID // Entry that was overridden
	WasFinal bool      // Whether the winner had the final flag
}

// Compose produces an OutputPlan from the given sources and entries.
// Sources must be ordered by scope (universal first).
// Entries belong to the listed sources.
// Compose is a pure function — no I/O, no side effects.
func Compose(consumer registry.Consumer, sources []registry.Source, entries []registry.Entry) (OutputPlan, error) {
	// Build source order map
	sourceOrder := make(map[uuid.UUID]int)
	for i, s := range sources {
		sourceOrder[s.ID] = i
	}

	// Group entries by capability type
	byType := make(map[registry.CapabilityType][]registry.Entry)
	for _, e := range entries {
		byType[e.Type] = append(byType[e.Type], e)
	}

	var plan OutputPlan
	plan.Consumer = consumer.ID

	for typ, typeEntries := range byType {
		conv := registry.Conventions[typ]

		switch effectivePrimitive(typeEntries, conv) {
		case registry.PrimitiveOverride:
			files, overrides := composeOverride(typeEntries, sourceOrder, conv)
			plan.Files = append(plan.Files, files...)
			plan.Overrides = append(plan.Overrides, overrides...)

		case registry.PrimitiveConcatenation:
			files := composeConcatenation(typeEntries, sourceOrder, conv)
			plan.Files = append(plan.Files, files...)

		case registry.PrimitiveReference:
			// Reference does not produce output files; it's a deployment mechanism
			// handled by the deployer.

		case registry.PrimitiveInheritance:
			// Inheritance is resolved at registry query time (universal → all consumers).
		}
	}

	return plan, nil
}

// effectivePrimitive returns the primitive to use for a set of entries.
// If any entry has a PrimitiveOverride, it wins. Otherwise, the convention default.
func effectivePrimitive(entries []registry.Entry, conv registry.Convention) registry.Primitive {
	for _, e := range entries {
		if e.PrimitiveOverride != nil {
			return *e.PrimitiveOverride
		}
	}
	return conv.DefaultPrimitive
}

// composeOverride resolves entries with the override primitive.
// Deeper scope wins unless a shallower entry has Final=true.
func composeOverride(entries []registry.Entry, sourceOrder map[uuid.UUID]int, conv registry.Convention) ([]OutputFile, []OverrideRecord) {
	// Group by destination path
	byPath := make(map[string][]registry.Entry)
	for _, e := range entries {
		path := convFileName(conv, e.Name)
		byPath[path] = append(byPath[path], e)
	}

	var files []OutputFile
	var overrides []OverrideRecord

	for path, pathEntries := range byPath {
		winner := resolveWinner(pathEntries, sourceOrder)
		// Record overrides
		for _, e := range pathEntries {
			if e.ID != winner.ID {
				overrides = append(overrides, OverrideRecord{
					Path:     path,
					Winner:   winner.ID,
					Loser:    e.ID,
					WasFinal: winner.Final,
				})
			}
		}
		files = append(files, OutputFile{
			DestinationPath: path,
			Primitive:       registry.PrimitiveOverride,
			SourceEntries:   []uuid.UUID{winner.ID},
		})
	}

	return files, overrides
}

// resolveWinner picks the winning entry for a path.
// Deeper scope wins. Final flag on a shallower entry prevents deeper override.
func resolveWinner(entries []registry.Entry, sourceOrder map[uuid.UUID]int) registry.Entry {
	winner := entries[0]
	for _, e := range entries[1:] {
		if winner.Final {
			// Winner is final — cannot be overridden
			continue
		}
		if sourceOrder[e.SourceID] > sourceOrder[winner.SourceID] {
			// Deeper scope wins
			winner = e
		}
	}
	return winner
}

// composeConcatenation concatenates entries in source order.
func composeConcatenation(entries []registry.Entry, sourceOrder map[uuid.UUID]int, conv registry.Convention) []OutputFile {
	if len(entries) == 0 {
		return nil
	}

	// Sort by source order
	sortBySourceOrder(entries, sourceOrder)

	var sourceIDs []uuid.UUID
	for _, e := range entries {
		sourceIDs = append(sourceIDs, e.ID)
	}

	path := convFileName(conv, entries[0].Name)
	return []OutputFile{{
		DestinationPath: path,
		Primitive:       registry.PrimitiveConcatenation,
		SourceEntries:   sourceIDs,
		// Content and ContentHash are filled by the deployer after reading files
	}}
}

// convFileName returns the conventional output path for an entry.
func convFileName(conv registry.Convention, name string) string {
	if conv.IsSingleton {
		return conv.SingletonName
	}
	fn := conv.FileName(name)
	if conv.Dir != "" {
		return conv.Dir + "/" + fn
	}
	return fn
}

func sortBySourceOrder(entries []registry.Entry, order map[uuid.UUID]int) {
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			if order[entries[i].SourceID] > order[entries[j].SourceID] {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}
}
