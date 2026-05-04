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
	Path  string       // Mutable, absolute path on disk
	Kind  ConsumerKind // Repo, Container, Host
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
	ContentHash       uint64         // xxHash of file content at last sync
	Final             bool           // If true, deeper scopes cannot override
	PrimitiveOverride *Primitive     // nil = use type default
	CreatedBy         EntryOrigin    // Carrel or UserAuthored
}

// Deployment records a deployment attempt.
type Deployment struct {
	ID          uuid.UUID
	ConsumerID  uuid.UUID
	AttemptedAt time.Time
	SucceededAt *time.Time // nil if attempt failed
	ConfigHash  uint64     // xxHash of the composed output plan
}

// DeploymentEntry records a deployed file from a deployment.
type DeploymentEntry struct {
	DeploymentID uuid.UUID
	Path         string    // Absolute destination path
	ContentHash  uint64    // xxHash of deployed content
	SourceEntry  uuid.UUID // FK to Entry that produced this
}
