package query

import "github.com/dichodaemon/carrel/internal/registry"

// QueryRegistry extends registry.Registry with query-specific methods.
type QueryRegistry interface {
	registry.Registry

	ListFiles(opts FileQueryOpts) ([]FileResult, error)
	ListDeployed(opts DeployQueryOpts) ([]DeployResult, error)
	ListDependents(sourceAlias string) ([]ConsumerResult, error)
	ListFeeds(consumerAlias string) ([]SourceResult, error)
	TraceSource(sourceAlias string, typ registry.CapabilityType, name string) ([]TraceResult, error)
	TraceDeployed(consumerAlias string, path string) ([]TraceResult, error)
	Plan(consumerAlias string) ([]PlanResult, error)
}
