package scanner

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/dichodaemon/carrel/internal/registry"
)

// DiscoveredRepo represents a repository found in the workspace.
type DiscoveredRepo struct {
	Path       string
	GitRemote  string
	HasCarrel  bool
	HasCarula  bool
	Registered bool
	ConsumerID *uuid.UUID
}

// Discover scans the workspace for git repositories and reports their status.
func Discover(reg registry.Registry, workspacePath string) ([]DiscoveredRepo, error) {
	entries, err := os.ReadDir(workspacePath)
	if err != nil {
		return nil, err
	}

	var repos []DiscoveredRepo
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		repoPath := filepath.Join(workspacePath, e.Name())
		if !isGitRepo(repoPath) {
			continue
		}

		dr := DiscoveredRepo{Path: repoPath}

		if _, err := os.Stat(filepath.Join(repoPath, ".carrel")); err == nil {
			dr.HasCarrel = true
		}

		if fi, err := os.Lstat(filepath.Join(repoPath, ".omp")); err == nil && fi.Mode()&os.ModeSymlink != 0 {
			dr.HasCarula = true
		}

		dr.GitRemote = readGitRemote(repoPath)

		c, err := reg.ResolveConsumer(repoPath)
		if err == nil {
			dr.Registered = true
			dr.ConsumerID = &c.ID
		}

		repos = append(repos, dr)
	}

	return repos, nil
}

// MigrationResult records the outcome of migrating a single carula repo.
type MigrationResult struct {
	Path       string
	Success    bool
	Error      error
	ConsumerID uuid.UUID
}

// Migrate imports carula-style .omp/ symlink targets into the registry.
func Migrate(reg registry.Registry, workspacePath string) ([]MigrationResult, error) {
	entries, err := os.ReadDir(workspacePath)
	if err != nil {
		return nil, err
	}

	var results []MigrationResult
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		repoPath := filepath.Join(workspacePath, e.Name())
		if !isGitRepo(repoPath) {
			continue
		}

		ompPath := filepath.Join(repoPath, ".omp")
		fi, err := os.Lstat(ompPath)
		if err != nil || fi.Mode()&os.ModeSymlink == 0 {
			continue
		}

		target, err := os.Readlink(ompPath)
		if err != nil {
			results = append(results, MigrationResult{Path: repoPath, Error: err})
			continue
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(repoPath, target)
		}

		alias := e.Name()
		c := registry.Consumer{
			ID:    uuid.New(),
			Alias: alias,
			Path:  repoPath,
			Kind:  registry.ConsumerRepo,
		}
		if err := reg.RegisterConsumer(c); err != nil {
			results = append(results, MigrationResult{Path: repoPath, Error: err})
			continue
		}

		s := registry.Source{
			ID:    uuid.New(),
			Alias: alias + "-omp",
			Path:  target,
			Scope: registry.ScopeTargetSpecific,
			Kind:  registry.SourceGitBacked,
		}
		_ = reg.RegisterSource(s)

		results = append(results, MigrationResult{
			Path:       repoPath,
			Success:    true,
			ConsumerID: c.ID,
		})
	}

	return results, nil
}

func isGitRepo(path string) bool {
	_, err := os.Stat(filepath.Join(path, ".git"))
	return err == nil
}

func readGitRemote(path string) string {
	data, err := os.ReadFile(filepath.Join(path, ".git", "config"))
	if err != nil {
		return ""
	}
	lines := strings.Split(string(data), "\n")
	inRemote := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "[remote \"origin\"]" {
			inRemote = true
			continue
		}
		if inRemote && strings.HasPrefix(line, "url = ") {
			return strings.TrimPrefix(line, "url = ")
		}
		if inRemote && strings.HasPrefix(line, "[") {
			inRemote = false
		}
	}
	return ""
}
