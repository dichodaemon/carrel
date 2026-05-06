package registry

import (
	"time"

	"github.com/google/uuid"
)

// ConsumerKind classifies the consumer type.
type ConsumerKind int

const (
	ConsumerRepo      ConsumerKind = iota // Target repository
	ConsumerContainer                      // Container (os-setup)
	ConsumerHost                           // Host machine (host-setup)
)

// Consumer represents a target that receives assembled configuration.
type Consumer struct {
	ID    uuid.UUID    // Immutable, assigned at registration
	Alias string       // Mutable, unique, initialized from path basename
	Path       string       // Mutable, absolute path on disk
	DeployRoot string       // Deployment root for this consumer (e.g., <path>/.omp, /home/dev)
	Kind       ConsumerKind // Repo, Container, Host
}

// SourceScope determines which consumers a source applies to.
type SourceScope int

const (
	ScopeUniversal      SourceScope = iota // Applies to all consumers
	ScopeTargetSpecific                     // Applies to one consumer
	ScopeUserPersonal                       // User OS tool customization
	ScopeHostLocal                          // Host-local overrides
)

// SourceKind classifies the source storage type.
type SourceKind int

const (
	SourceGitBacked SourceKind = iota
	SourceHostLocal
)

// Source is a directory or repository providing configuration entries.
type Source struct {
	ID    uuid.UUID   // Immutable
	Alias string      // Mutable, unique
	Path  string      // Mutable, absolute path on disk
	Scope SourceScope // Universal, TargetSpecific, UserPersonal, HostLocal
	Kind  SourceKind  // GitBacked, HostLocal
}

// CapabilityType enumerates OMP and OS tool configuration types.
type CapabilityType int

const (
	TypeRule CapabilityType = iota
	TypeSkill
	TypeCommand
	TypeExtension
	TypeAgent
	TypeTool
	TypeHook
	TypePrompt
	TypeInstruction
	TypeContextFile
	TypeAppendSystem
	// OS tool types
	TypeZshConfig
	TypeNvimConfig
	TypeWeztermConfig
	TypeP10kConfig
)

// Primitive defines the composition operation.
type Primitive int

const (
	PrimitiveOverride      Primitive = iota
	PrimitiveConcatenation
	PrimitiveReference
	PrimitiveInheritance
)

// ComposeMode is an alias for Primitive for use in entries and slots.
// It carries the same values (Override=0, Concatenation=1) but is named
// to reflect its role as a composition operator, not a general primitive.
type ComposeMode = Primitive

const (
	ModeOverride      ComposeMode = PrimitiveOverride
	ModeConcatenation ComposeMode = PrimitiveConcatenation
)

// EntryOrigin records who created the entry.
type EntryOrigin int

const (
	OriginCarrel      EntryOrigin = iota // Created by carrel <type> add
	OriginUserAuthored                    // Discovered by carrel, authored by user
)

// Entry is a single configuration artifact in the registry.
type Entry struct {
	ID                uuid.UUID      // Immutable
	SourceID          uuid.UUID      // FK to Source
	Name              string         // Entry name (e.g., "no-pushing-master")
	Type              CapabilityType // Rule, Skill, Command, etc.
	RelativePath      string         // Path relative to source root
	ContentHash       int64          // xxHash of file content at last sync
	Final             bool           // If true, deeper scopes cannot override
	ComposeMode       ComposeMode    // override or concatenation (mandatory, set by scan from convention)
	CreatedBy         EntryOrigin    // Carrel or UserAuthored
}

// Deployment records a deployment attempt.
type Deployment struct {
	ID          uuid.UUID
	ConsumerID  uuid.UUID
	AttemptedAt time.Time
	SucceededAt *time.Time // nil if attempt failed
	ConfigHash  int64     // xxHash of the composed output plan
}

// DeploymentEntry records a deployed file from a deployment.
type DeploymentEntry struct {
	DeploymentID uuid.UUID
	Path         string    // Absolute destination path
	ContentHash  int64     // xxHash of deployed content
	SourceEntry  uuid.UUID // FK to Entry that produced this
	SlotID       uuid.UUID // FK to Slot that produced this (v3)
}

// Slot is an output target for a specific consumer.
type Slot struct {
	ID          uuid.UUID
	ConsumerID  uuid.UUID
	Name        string       // Human-readable name, unique per consumer
	DestPath    string       // Relative path from consumer's deploy_root
	ComposeMode *ComposeMode // nil = use entry-level modes; set = override
}

// SlotUpdates carries optional changes for UpdateSlot.
type SlotUpdates struct {
	Name        *string
	DestPath    *string
	ComposeMode **ComposeMode // nil = no change; *nil = clear; set = override
}

// EntrySlot maps an entry to a slot (N:M) with a priority.
type EntrySlot struct {
	EntryID  uuid.UUID
	SlotID   uuid.UUID
	Priority int
}

// SlotEntry is an Entry annotated with its slot membership priority.
type SlotEntry struct {
	Entry
	Priority int
}
