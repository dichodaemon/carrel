package deployer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cespare/xxhash/v2"
	"github.com/dichodaemon/carrel/internal/composer"
	"github.com/dichodaemon/carrel/internal/registry"
)

// ConflictPolicy controls how the deployer handles files not owned by carrel.
type ConflictPolicy int

const (
	ConflictError     ConflictPolicy = iota // stop and return error
	ConflictBackup                          // rename existing to .bak, then write
	ConflictSkip                            // leave existing file in place
	ConflictOverwrite                       // overwrite foreign file in place
)

// ConflictAction records what happened to a foreign file during deployment.
type ConflictAction int

const (
	ActionReplaced   ConflictAction = iota // carrel-owned file, replaced
	ActionBackedUp                         // foreign file, renamed to .bak
	ActionSkipped                          // foreign file, skipped
	ActionErrored                          // foreign file, would have caused error (dryRun only)
	ActionOverwritten                      // foreign file, overwritten in place
)

// CollisionResult records a single collision encountered during deployment.
type CollisionResult struct {
	Path        string
	ForeignHash int64
	Action      ConflictAction
}

// Deploy writes output files to disk, handling collisions and cleaning up stale
// files from a previous deployment.
//
// plan is the composed output from composer.Compose.
// previousClaim is the set of DeploymentEntry records from the last successful deployment.
// policy determines how foreign files are handled.
// dryRun records what would happen without writing any files.
func Deploy(
	plan composer.OutputPlan,
	previousClaim []registry.DeploymentEntry,
	policy ConflictPolicy,
	dryRun bool,
) ([]CollisionResult, error) {
	prevByPath := make(map[string]registry.DeploymentEntry, len(previousClaim))
	for _, e := range previousClaim {
		prevByPath[e.Path] = e
	}

	newPaths := make(map[string]bool, len(plan.Files))
	for _, f := range plan.Files {
		newPaths[f.DestinationPath] = true
	}

	var collisions []CollisionResult
	type writeOp struct {
		file      composer.OutputFile
		action    ConflictAction
		needsBackup bool
	}

	var ops []writeOp

	// --- Phase 1: validate every file ---
	for _, f := range plan.Files {
		stat, err := os.Stat(f.DestinationPath)
		if err != nil {
			if os.IsNotExist(err) {
				ops = append(ops, writeOp{file: f})
				continue
			}
			return nil, fmt.Errorf("stat %s: %w", f.DestinationPath, err)
		}

		if stat.IsDir() {
			return nil, fmt.Errorf("%s is a directory, not a file", f.DestinationPath)
		}

		existingData, err := os.ReadFile(f.DestinationPath)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", f.DestinationPath, err)
		}
		existingHash := int64(xxhash.Sum64(existingData))

		prev, isClaimed := prevByPath[f.DestinationPath]
		if isClaimed && prev.ContentHash == existingHash {
			// Carrel-owned and hash matches — safe to replace.
			ops = append(ops, writeOp{file: f, action: ActionReplaced})
			continue
		}

		// Foreign or tampered.
		action := resolveConflict(policy)
		collisions = append(collisions, CollisionResult{
			Path:        f.DestinationPath,
			ForeignHash: existingHash,
			Action:      action,
		})

		switch action {
		case ActionErrored:
			if !dryRun {
				return collisions, fmt.Errorf("conflict at %s: foreign file exists", f.DestinationPath)
			}
			continue
		case ActionSkipped:
			continue
		case ActionBackedUp:
			ops = append(ops, writeOp{file: f, action: ActionBackedUp, needsBackup: true})
		case ActionOverwritten:
			ops = append(ops, writeOp{file: f, action: ActionOverwritten})
		}
	}

	if dryRun {
		// Phase 2 (staleness) still produces collision records even for dryRun.
		staleCollisions := cleanupStale(prevByPath, newPaths, dryRun)
		collisions = append(collisions, staleCollisions...)
		return collisions, nil
	}

	// --- Phase 2: write files ---
	for _, op := range ops {
		if op.needsBackup {
			backupPath := op.file.DestinationPath + ".bak"
			if err := os.Rename(op.file.DestinationPath, backupPath); err != nil {
				return collisions, fmt.Errorf("backup %s → %s: %w", op.file.DestinationPath, backupPath, err)
			}
		}

		dir := filepath.Dir(op.file.DestinationPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return collisions, fmt.Errorf("mkdir %s: %w", dir, err)
		}

		if err := os.WriteFile(op.file.DestinationPath, op.file.Content, 0644); err != nil {
			return collisions, fmt.Errorf("write %s: %w", op.file.DestinationPath, err)
		}
	}

	// --- Phase 3: cleanup stale files ---
	staleCollisions := cleanupStale(prevByPath, newPaths, dryRun)
	collisions = append(collisions, staleCollisions...)

	return collisions, nil
}

// resolveConflict maps a ConflictPolicy to a ConflictAction.
func resolveConflict(policy ConflictPolicy) ConflictAction {
	switch policy {
	case ConflictError:
		return ActionErrored
	case ConflictBackup:
		return ActionBackedUp
	case ConflictSkip:
		return ActionSkipped
	case ConflictOverwrite:
		return ActionOverwritten
	default:
		return ActionSkipped
	}
}

// cleanupStale removes files from previousClaim that are not in newPaths.
// Files whose on-disk hash matches the claim are removed.
// Files whose hash has changed are recorded as collisions and left in place.
func cleanupStale(
	prevByPath map[string]registry.DeploymentEntry,
	newPaths map[string]bool,
	dryRun bool,
) []CollisionResult {
	var collisions []CollisionResult

	for path, prev := range prevByPath {
		if newPaths[path] {
			continue
		}

		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			// Read error: record and skip — can't safely remove.
			collisions = append(collisions, CollisionResult{
				Path:        path,
				ForeignHash: 0,
				Action:      ActionSkipped,
			})
			continue
		}

		actualHash := int64(xxhash.Sum64(data))
		if actualHash == prev.ContentHash {
			if !dryRun {
				if err := os.Remove(path); err != nil {
					collisions = append(collisions, CollisionResult{
						Path:        path,
						ForeignHash: actualHash,
						Action:      ActionSkipped,
					})
				}
			}
		} else {
			collisions = append(collisions, CollisionResult{
				Path:        path,
				ForeignHash: actualHash,
				Action:      ActionSkipped,
			})
		}
	}

	return collisions
}

// EnsureGitExclude adds .omp/ and any deployed paths outside it to
// .git/info/exclude for non-opt-in repos. deployedPaths are absolute
// destination paths from the deployment plan; deployRoot is the consumer's
// deploy root (e.g., /workspace/core-stack/.omp).
func EnsureGitExclude(repoPath string, deployRoot string, deployedPaths []string) error {
	// Only add exclude for repos without .carrel opt-in
	if _, err := os.Stat(filepath.Join(repoPath, ".carrel")); err == nil {
		return nil // opt-in repo, skip
	}

	excludePath := filepath.Join(repoPath, ".git", "info", "exclude")

	// Create .git/info directory if missing
	if err := os.MkdirAll(filepath.Dir(excludePath), 0755); err != nil {
		return err
	}

	// Read existing exclude content
	existing, _ := os.ReadFile(excludePath)
	existingStr := string(existing)

	// Base patterns: the deploy root directory itself
	patterns := []string{".omp/", "AGENTS.md"}

	// Add any deployed path that lands outside the deploy root
	cleanRoot := filepath.Clean(deployRoot)
	for _, p := range deployedPaths {
		cleanP := filepath.Clean(p)
		if !strings.HasPrefix(cleanP, cleanRoot+"/") && cleanP != cleanRoot {
			// Path is outside deploy root — make it relative to the repo
			rel, err := filepath.Rel(repoPath, cleanP)
			if err != nil {
				continue
			}
			patterns = append(patterns, rel)
		}
	}

	var toAdd []string
	for _, p := range patterns {
		if !strings.Contains(existingStr, p) {
			toAdd = append(toAdd, p)
		}
	}
	if len(toAdd) == 0 {
		return nil
	}

	// Append new patterns
	f, err := os.OpenFile(excludePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, p := range toAdd {
		if _, err := fmt.Fprintf(f, "%s\n", p); err != nil {
			return err
		}
	}
	return nil
}
