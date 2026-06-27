package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func parentFetchSmokeDir(t *testing.T) string {
	t.Helper()
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "testdata", "parent_fetch_smoke")
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("parent_fetch_smoke dir missing: %v", err)
	}
	return dir
}

func runParentFetchSmokeScript(t *testing.T, dir string) string {
	t.Helper()
	if _, err := exec.LookPath("resty"); err != nil {
		return runParentFetchSmokeDocker(t, dir)
	}
	script := filepath.Join(dir, "run_scenarios.sh")
	cmd := exec.Command("sh", script)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(out), "Cannot connect to the Docker daemon") {
			t.Skip("docker unavailable")
		}
		t.Fatalf("parent fetch smoke script failed: %v\n%s", err, out)
	}
	return string(out)
}

func runParentFetchSmokeDocker(t *testing.T, dir string) string {
	t.Helper()
	build := exec.Command("docker", "compose", "-f", filepath.Join(dir, "docker-compose.yml"), "build", "--quiet")
	build.Dir = dir
	if out, err := build.CombinedOutput(); err != nil {
		if strings.Contains(string(out), "Cannot connect to the Docker daemon") {
			t.Skip("docker unavailable")
		}
		t.Fatalf("docker compose build failed: %v\n%s", err, out)
	}
	run := exec.Command("docker", "compose", "-f", filepath.Join(dir, "docker-compose.yml"), "run", "--rm", "parent-fetch-smoke")
	run.Dir = dir
	out, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("docker compose run failed: %v\n%s", err, out)
	}
	return string(out)
}

func TestParentFetchAccessGuardLuaScenarios(t *testing.T) {
	dir := parentFetchSmokeDir(t)
	out := runParentFetchSmokeScript(t, dir)
	if !strings.Contains(out, "ALL_SCENARIOS_OK") {
		t.Fatalf("expected ALL_SCENARIOS_OK, got:\n%s", out)
	}
	for _, name := range []string{
		"L3_origin",
		"L3_l2",
		"L3_l1",
		"L1_use_l2_healthy",
		"L1_skip_l2_header",
		"L3_l1_failover_l2",
		"L3_l1_both_down_origin",
		"L3_l2_down_origin",
		"L3_l1_recovery",
		"L1_respect_l2_parse",
	} {
		if !strings.Contains(out, "PASS "+name) {
			t.Fatalf("missing scenario %s in output:\n%s", name, out)
		}
	}
}

func TestParentFetchDockerComposeSmoke(t *testing.T) {
	if os.Getenv("SKIP_DOCKER_COMPOSE") == "1" {
		t.Skip("SKIP_DOCKER_COMPOSE=1")
	}
	dir := parentFetchSmokeDir(t)
	build := exec.Command("docker", "compose", "-f", filepath.Join(dir, "docker-compose.yml"), "build", "--quiet")
	build.Dir = dir
	if out, err := build.CombinedOutput(); err != nil {
		if strings.Contains(string(out), "Cannot connect to the Docker daemon") {
			t.Skip("docker unavailable")
		}
		t.Fatalf("docker compose build failed: %v\n%s", err, out)
	}
	run := exec.Command("docker", "compose", "-f", filepath.Join(dir, "docker-compose.yml"), "run", "--rm", "parent-fetch-smoke")
	run.Dir = dir
	out, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("docker compose run failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "ALL_SCENARIOS_OK") {
		t.Fatalf("expected ALL_SCENARIOS_OK from compose run, got:\n%s", out)
	}
}

func TestParentFetchOpenRestyDockerResty(t *testing.T) {
	dir := parentFetchSmokeDir(t)
	luaDir := filepath.Join(dir, "lua")
	scriptPath := filepath.Join(dir, "run_scenarios.sh")
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatal(err)
	}
	restyBlock := strings.Split(string(content), "resty -e \"")
	if len(restyBlock) < 2 {
		t.Fatal("unexpected run_scenarios.sh format")
	}
	luaScript := strings.TrimSuffix(restyBlock[1], "\"\n")
	cmd := exec.Command("docker", "run", "--rm",
		"-v", luaDir+":/lua:ro",
		"openresty/openresty:1.25.3.1-0-alpine-fat",
		"resty", "-e", strings.ReplaceAll(luaScript, "$ROOT/lua/?.lua", "/lua/?.lua"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(out), "Cannot connect to the Docker daemon") {
			t.Skip("docker unavailable")
		}
		t.Fatalf("openresty docker resty failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "ALL_SCENARIOS_OK") {
		t.Fatalf("unexpected output: %s", out)
	}
}
