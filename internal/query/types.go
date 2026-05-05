package query

import "github.com/dichodaemon/carrel/internal/registry"

// FileResult represents a source file in the registry with its on-disk status.
type FileResult struct {
	SourceAlias string              `json:"sourceAlias"`
	Type        registry.CapabilityType `json:"type"`
	TypeName    string              `json:"typeName"`
	Name        string              `json:"name"`
	Path        string              `json:"path"`
	Status      FileStatus          `json:"status"`
	ContentHash uint64              `json:"contentHash"`
	Final       bool                `json:"final"`
	ID          string              `json:"id"`
}

type FileStatus int

const (
	FileExists  FileStatus = iota
	FileMissing
)

func (s FileStatus) String() string {
	switch s {
	case FileExists:
		return "EXISTS"
	case FileMissing:
		return "MISSING"
	}
	return "UNKNOWN"
}

// FileQueryOpts filters the result of ListFiles.
type FileQueryOpts struct {
	Type     *registry.CapabilityType
	Source   *string
	Valid    bool
	Orphaned bool
}

// DeployResult represents a deployed file with its verification status.
type DeployResult struct {
	ConsumerAlias string       `json:"consumerAlias"`
	Path          string       `json:"path"`
	Status        DeployStatus `json:"status"`
	ClaimHash     uint64       `json:"claimHash"`
	ActualHash    uint64       `json:"actualHash"`
	ID            string       `json:"id"`
}

type DeployStatus int

const (
	DeployDeployed  DeployStatus = iota
	DeployModified
	DeployMissing
	DeployForeign
)

func (s DeployStatus) String() string {
	switch s {
	case DeployDeployed:
		return "DEPLOYED"
	case DeployModified:
		return "MODIFIED"
	case DeployMissing:
		return "MISSING"
	case DeployForeign:
		return "FOREIGN"
	}
	return "UNKNOWN"
}

// DeployQueryOpts filters the result of ListDeployed.
type DeployQueryOpts struct {
	Consumer *string
	Claims   bool
	Actual   bool
}

// PlanResult represents a planned deployment change.
type PlanResult struct {
	ConsumerAlias string     `json:"consumerAlias"`
	Path          string     `json:"path"`
	Change        PlanChange `json:"change"`
	ClaimHash     uint64     `json:"claimHash"`
	PlanHash      uint64     `json:"planHash"`
}

type PlanChange int

const (
	PlanAdded    PlanChange = iota
	PlanModified
	PlanRemoved
)

func (c PlanChange) String() string {
	switch c {
	case PlanAdded:
		return "+ added"
	case PlanModified:
		return "~ modified"
	case PlanRemoved:
		return "- removed"
	}
	return "?"
}

// ConsumerResult represents a consumer in dependency queries.
type ConsumerResult struct {
	Alias string `json:"alias"`
	Path  string `json:"path"`
}

// SourceResult represents a source in dependency queries.
type SourceResult struct {
	Alias string              `json:"alias"`
	Path  string              `json:"path"`
	Scope registry.SourceScope `json:"scope"`
}

// TraceResult represents a composition provenance link.
type TraceResult struct {
	ConsumerAlias string              `json:"consumerAlias,omitempty"`
	DeployedPath  string              `json:"deployedPath,omitempty"`
	SourceAlias   string              `json:"sourceAlias,omitempty"`
	SourceType    registry.CapabilityType `json:"sourceType,omitempty"`
	SourceName    string              `json:"sourceName,omitempty"`
	SourcePath    string              `json:"sourcePath,omitempty"`
}
