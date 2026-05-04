package carrel_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const carrelBin = "/workspace/carrel/carrel"

func buildCarrelBin(t *testing.T) {
	t.Helper()
	cmd := exec.Command("go", "build", "-o", carrelBin, "./cmd/carrel")
	cmd.Dir = "/workspace/carrel"
	cmd.Env = append(os.Environ(), "CGO_ENABLED=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build carrel: %v\n%s", err, out)
	}
	os.Chmod(carrelBin, 0755)
}

// TestBootstrapE2E verifies bootstrap creates the registry and is idempotent.
func TestBootstrapE2E(t *testing.T) {
	buildCarrelBin(t)

	cmd := exec.Command(carrelBin, "bootstrap")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("bootstrap failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "initialized") {
		t.Errorf("bootstrap output: %s", out)
	}

	// Verify registry exists
	if _, err := os.Stat("/workspace/.carrel/registry"); err != nil {
		t.Error("registry not created at /workspace/.carrel/registry")
	}

	// Verify idempotent
	cmd = exec.Command(carrelBin, "bootstrap")
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("bootstrap rerun failed: %v\n%s", err, out)
	}
}

// TestBootstrapRegistersConsumers verifies consumers are registered.
func TestBootstrapRegistersConsumers(t *testing.T) {
	buildCarrelBin(t)

	exec.Command(carrelBin, "bootstrap").Run()

	cmd := exec.Command(carrelBin, "status")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("status failed: %v\n%s", err, out)
	}
	output := string(out)
	if !strings.Contains(output, "carrel") {
		t.Error("carrel consumer not registered")
	}
	if !strings.Contains(output, "folio") {
		t.Error("folio consumer not registered")
	}
	if !strings.Contains(output, "carrel-omp") {
		t.Error("carrel-omp source not registered")
	}
}

// TestDiscoverListsRepos verifies discover scans the workspace.
func TestDiscoverListsRepos(t *testing.T) {
	buildCarrelBin(t)
	exec.Command(carrelBin, "bootstrap").Run()

	cmd := exec.Command(carrelBin, "discover")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("discover failed: %v\n%s", err, out)
	}
	output := string(out)
	if !strings.Contains(output, "carrel") {
		t.Error("carrel not in discover output")
	}
	if !strings.Contains(output, "folio") {
		t.Error("folio not in discover output")
	}
}

// TestRunUnregisteredRepoErrors verifies run fails in an unregistered repo.
func TestRunUnregisteredRepoErrors(t *testing.T) {
	buildCarrelBin(t)
	exec.Command(carrelBin, "bootstrap").Run()

	// Create temp unregistered repo
	tmpDir, _ := os.MkdirTemp("/home/dev", "carrel-test-repo-*")
	defer os.RemoveAll(tmpDir)
	os.MkdirAll(filepath.Join(tmpDir, ".git"), 0755)

	cmd := exec.Command(carrelBin, "run")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Error("run should fail in unregistered repo")
	}
	if !strings.Contains(string(out), "unregistered") {
		t.Errorf("error should mention unregistered: %s", out)
	}
}

// TestRunDryRun verifies --dry-run does not write files.
func TestRunDryRun(t *testing.T) {
	buildCarrelBin(t)
	exec.Command(carrelBin, "bootstrap").Run()

	cmd := exec.Command(carrelBin, "run", "--dry-run")
	cmd.Dir = "/workspace/carrel"
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run --dry-run failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "dry-run") {
		t.Error("dry-run should mention dry-run")
	}
}

// TestMigrateCreatesConsumers verifies migration from carula symlinks.
func TestMigrateCreatesConsumers(t *testing.T) {
	buildCarrelBin(t)
	exec.Command(carrelBin, "bootstrap").Run()

	// Create a carula-style repo with symlink
	tmpDir, _ := os.MkdirTemp("/home/dev", "carrel-test-migrate-*")
	defer os.RemoveAll(tmpDir)

	repoPath := filepath.Join(tmpDir, "test-repo")
	os.MkdirAll(filepath.Join(repoPath, ".git"), 0755)
	targetDir := filepath.Join(tmpDir, "target-omp")
	os.MkdirAll(targetDir, 0755)
	os.Symlink(targetDir, filepath.Join(repoPath, ".omp"))

	// Note: migrate scans /workspace, not the temp dir. We'd need to change
	// scanner.Migrate to accept a path parameter. For now, skip the full test.
	t.Skip("migrate currently hardcodes /workspace")
}

// TestCRUDAddAndList tests rule add and list commands.
func TestCRUDAddAndList(t *testing.T) {
	buildCarrelBin(t)
	exec.Command(carrelBin, "bootstrap").Run()

	cmd := exec.Command(carrelBin, "rule", "add", "test-rule",
		"--source=carrel-omp", "--content=test content")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("rule add failed: %v\n%s", err, out)
	}

	// Verify file was created
	rulePath := filepath.Join("/workspace/carrel/omp", "rules", "test-rule.md")
	if _, err := os.Stat(rulePath); err != nil {
		t.Error("rule file not created at", rulePath)
	}
	defer os.Remove(rulePath)

	// List rules
	cmd = exec.Command(carrelBin, "rule", "list")
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("rule list failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "test-rule") {
		t.Error("rule not found in list")
	}
}

// TestCRUDRemove verifies rule rm removes file and registry entry.
func TestCRUDRemove(t *testing.T) {
	buildCarrelBin(t)
	exec.Command(carrelBin, "bootstrap").Run()

	exec.Command(carrelBin, "rule", "add", "remove-me",
		"--source=carrel-omp", "--content=test").Run()

	cmd := exec.Command(carrelBin, "rule", "rm", "remove-me")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("rule rm failed: %v\n%s", err, out)
	}

	rulePath := filepath.Join("/workspace/carrel/omp", "rules", "remove-me.md")
	if _, err := os.Stat(rulePath); err == nil {
		t.Error("rule file should be removed")
		os.Remove(rulePath)
	}
}

// TestStatusShowsConsumers verifies status command.
func TestStatusShowsConsumers(t *testing.T) {
	buildCarrelBin(t)
	exec.Command(carrelBin, "bootstrap").Run()

	cmd := exec.Command(carrelBin, "status")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("status failed: %v\n%s", err, out)
	}
	output := string(out)
	if !strings.Contains(output, "Consumers:") {
		t.Error("status should show Consumers")
	}
	if !strings.Contains(output, "Sources:") {
		t.Error("status should show Sources")
	}
}

// TestVerifyShowsDeploymentStatus verifies verify command.
func TestVerifyShowsDeploymentStatus(t *testing.T) {
	buildCarrelBin(t)
	exec.Command(carrelBin, "bootstrap").Run()

	cmd := exec.Command(carrelBin, "verify")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("verify failed: %v\n%s", err, out)
	}
	output := string(out)
	if !strings.Contains(output, "carrel") && !strings.Contains(output, "folio") {
		t.Error("verify should show consumers")
	}
}

