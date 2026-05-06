package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cespare/xxhash/v2"
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

// ScanResult records the outcome of scanning for one entry.
type ScanResult struct {
	Name   string
	Type   string // human-readable type name
	Action string // "registered", "skipped", or "error"
	Error  error
}

// ScanSource walks a source directory and registers any files matching
// capability type conventions. Already-registered entries are skipped.
func ScanSource(reg registry.Registry, source registry.Source) ([]ScanResult, error) {
	entries, err := reg.ResolveEntries([]uuid.UUID{source.ID})
	if err != nil {
		return nil, fmt.Errorf("resolve entries: %w", err)
	}
	existing := make(map[string]bool)
	for _, e := range entries {
		existing[fmt.Sprintf("%d:%s", e.Type, e.Name)] = true
	}

	var results []ScanResult

	for typ, conv := range registry.Conventions {
		if conv.Dir != "" {
			typeDir := filepath.Join(source.Path, conv.Dir)
			if info, err := os.Stat(typeDir); err != nil || !info.IsDir() {
				continue
			}

			if typ == registry.TypeSkill {
				results = append(results, scanSkillDir(reg, source, typ, typeDir, conv, existing)...)
			} else if conv.IsSingleton {
				results = append(results, scanSingleton(reg, source, typ, typeDir, conv, existing)...)
			} else {
				// Pass singleton file names for this directory so scanFileDir can exclude them
				singles := singletonFiles(typeDir)
				results = append(results, scanFileDir(reg, source, typ, typeDir, conv, existing, singles)...)
			}
		} else if conv.IsSingleton {
			results = append(results, scanSingleton(reg, source, typ, source.Path, conv, existing)...)
		}
	}

	return results, nil
}

// singletonFiles returns the set of filenames that are claimed by singleton
// conventions in the same directory (e.g., p10k.zsh in config/zsh).
func singletonFiles(dir string) map[string]bool {
	out := make(map[string]bool)
	for _, conv := range registry.Conventions {
		if conv.IsSingleton && conv.Dir != "" {
			sDir := conv.Dir
			if strings.HasSuffix(dir, "/"+sDir) || dir == sDir {
				out[conv.SingletonName] = true
			}
		}
	}
	return out
}

func scanFileDir(reg registry.Registry, source registry.Source, typ registry.CapabilityType, dir string, conv registry.Convention, existing map[string]bool, singletons map[string]bool) []ScanResult {
	dummy := conv.FileName("_")
	suffix := strings.TrimPrefix(dummy, "_")

	files, err := os.ReadDir(dir)
	if err != nil {
		return []ScanResult{{Name: dir, Type: "", Action: "error", Error: err}}
	}

	var results []ScanResult
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), suffix) {
			continue
		}
		if singletons[f.Name()] {
			continue // claimed by a singleton convention
		}
		name := strings.TrimSuffix(f.Name(), suffix)
		key := fmt.Sprintf("%d:%s", typ, name)
		if existing[key] {
			results = append(results, ScanResult{Name: name, Type: convTypeName(typ), Action: "skipped"})
			continue
		}

		fullPath := filepath.Join(dir, f.Name())
		content, err := os.ReadFile(fullPath)
		if err != nil {
			results = append(results, ScanResult{Name: name, Type: convTypeName(typ), Action: "error", Error: err})
			continue
		}

		relPath := conv.Dir + "/" + f.Name()
		entry := registry.Entry{
			ID:           uuid.New(),
			SourceID:     source.ID,
			Name:         name,
			Type:         typ,
			RelativePath: relPath,
			ContentHash:  int64(xxhash.Sum64(content)),
			ComposeMode:  registry.ComposeMode(conv.DefaultPrimitive),
			CreatedBy:    registry.OriginCarrel,
		}
		if err := reg.RegisterEntry(entry); err != nil {
			results = append(results, ScanResult{Name: name, Type: convTypeName(typ), Action: "error", Error: err})
			continue
		}
		results = append(results, ScanResult{Name: name, Type: convTypeName(typ), Action: "registered"})
	}
	return results
}

func scanSkillDir(reg registry.Registry, source registry.Source, typ registry.CapabilityType, skillsDir string, conv registry.Convention, existing map[string]bool) []ScanResult {
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return []ScanResult{{Name: skillsDir, Type: "", Action: "error", Error: err}}
	}

	var results []ScanResult
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		key := fmt.Sprintf("%d:%s", typ, name)
		if existing[key] {
			results = append(results, ScanResult{Name: name, Type: convTypeName(typ), Action: "skipped"})
			continue
		}

		skillFile := filepath.Join(skillsDir, name, "SKILL.md")
		content, err := os.ReadFile(skillFile)
		if err != nil {
			results = append(results, ScanResult{Name: name, Type: convTypeName(typ), Action: "error", Error: err})
			continue
		}

		relPath := conv.Dir + "/" + name + "/SKILL.md"
		entry := registry.Entry{
			ID:           uuid.New(),
			SourceID:     source.ID,
			Name:         name,
			Type:         typ,
			RelativePath: relPath,
			ContentHash:  int64(xxhash.Sum64(content)),
			ComposeMode:  registry.ComposeMode(conv.DefaultPrimitive),
			CreatedBy:    registry.OriginCarrel,
		}
		if err := reg.RegisterEntry(entry); err != nil {
			results = append(results, ScanResult{Name: name, Type: convTypeName(typ), Action: "error", Error: err})
			continue
		}
		results = append(results, ScanResult{Name: name, Type: convTypeName(typ), Action: "registered"})
	}
	return results
}

func scanSingleton(reg registry.Registry, source registry.Source, typ registry.CapabilityType, dir string, conv registry.Convention, existing map[string]bool) []ScanResult {
	name := conv.SingletonName
	filePath := filepath.Join(dir, name)
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}

	key := fmt.Sprintf("%d:%s", typ, name)
	if existing[key] {
		return []ScanResult{{Name: name, Type: convTypeName(typ), Action: "skipped"}}
	}

	relPath := name
	if conv.Dir != "" {
		relPath = conv.Dir + "/" + name
	}
	entry := registry.Entry{
		ID:           uuid.New(),
		SourceID:     source.ID,
		Name:         name,
		Type:         typ,
		RelativePath: relPath,
		ContentHash:  int64(xxhash.Sum64(content)),
		ComposeMode:  registry.ComposeMode(conv.DefaultPrimitive),
		CreatedBy:    registry.OriginCarrel,
	}
	if err := reg.RegisterEntry(entry); err != nil {
		return []ScanResult{{Name: name, Type: convTypeName(typ), Action: "error", Error: err}}
	}
	return []ScanResult{{Name: name, Type: convTypeName(typ), Action: "registered"}}
}

func convTypeName(typ registry.CapabilityType) string {
	switch typ {
	case registry.TypeRule:          return "rule"
	case registry.TypeSkill:         return "skill"
	case registry.TypeCommand:       return "command"
	case registry.TypeExtension:     return "extension"
	case registry.TypeAgent:         return "agent"
	case registry.TypeTool:          return "tool"
	case registry.TypeHook:          return "hook"
	case registry.TypePrompt:        return "prompt"
	case registry.TypeInstruction:   return "instruction"
	case registry.TypeContextFile:   return "context-file"
	case registry.TypeAppendSystem:  return "append-system"
	case registry.TypeZshConfig:     return "zsh"
	case registry.TypeNvimConfig:    return "nvim"
	case registry.TypeWeztermConfig: return "wezterm"
	case registry.TypeP10kConfig:    return "p10k"
	}
	return fmt.Sprintf("type-%d", typ)
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
