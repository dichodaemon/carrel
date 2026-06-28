package registry

import "github.com/google/uuid"

// Registry is the interface for the configuration store.
// Backed by Dolt in production, in-memory in tests.
type Registry interface {
	// Consumer operations
	RegisterConsumer(consumer Consumer) error
	ResolveConsumer(aliasOrPath string) (Consumer, error)
	ListConsumers() ([]Consumer, error)

	// Source operations
	RegisterSource(source Source) error
	ResolveSources(consumerID uuid.UUID) ([]Source, error)
	LinkConsumerSource(consumerID, sourceID uuid.UUID) error
	ListSources() ([]Source, error)

	// Entry operations
	RegisterEntry(entry Entry) error
	ResolveEntries(sourceIDs []uuid.UUID) ([]Entry, error)
	RemoveEntry(entryID uuid.UUID) error

	// Deployment operations
	RecordDeployment(deployment Deployment, entries []DeploymentEntry) error
	LastDeployment(consumerID uuid.UUID) (Deployment, []DeploymentEntry, error)

	// Slot operations
	RegisterSlot(slot Slot) error
	UpdateSlot(slotID uuid.UUID, updates SlotUpdates) error
	RemoveSlot(slotID uuid.UUID) error
	ResolveSlots(consumerID uuid.UUID) ([]Slot, error)

	// Entry-slot operations
	LinkEntrySlot(entryID, slotID uuid.UUID, priority int) error
	UnlinkEntrySlot(entryID, slotID uuid.UUID) error
	UnlinkAllEntrySlots(entryID uuid.UUID) error
	ResolveEntrySlots(slotID uuid.UUID) ([]SlotEntry, error)

	// Lifecycle
	Close() error
}


// Common errors.
var (
	ErrNotFound    = &RegistryError{"not found"}
	ErrDuplicate   = &RegistryError{"duplicate entry"}
	ErrAmbiguous   = &RegistryError{"ambiguous match"}
	ErrNoDeployment = &RegistryError{"no successful deployment recorded"}
)

// RegistryError is a simple error type for registry operations.
type RegistryError struct {
	Message string
}

func (e *RegistryError) Error() string { return e.Message }
