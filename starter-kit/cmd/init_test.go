package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CdeT-Comite-de-Transformacion/los-bff/starter-kit/internal/config"
)

func TestInitCreatesGoProjectSkeleton(t *testing.T) {
	dir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chdir(oldWd)
	}()

	cfg := config.Default()
	cfg.Project.Name = "demoapp"
	cfg.Project.Module = "example.com/demoapp"

	if err := initProject(cfg); err != nil {
		t.Fatalf("initProject returned error: %v", err)
	}
	if err := config.Save(config.DefaultFilename, cfg); err != nil {
		t.Fatalf("save config returned error: %v", err)
	}

	checks := []string{"modkit.yaml", "go.mod", "main.go", filepath.Join("cmd", "root.go")}
	for _, item := range checks {
		if _, err := os.Stat(item); err != nil {
			t.Fatalf("expected %s to exist: %v", item, err)
		}
	}

	modData, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(modData), "module example.com/demoapp") {
		t.Fatalf("go.mod should contain module example.com/demoapp, got: %s", string(modData))
	}

	mainData, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(mainData), "package main") {
		t.Fatalf("main.go should contain package main, got: %s", string(mainData))
	}
}
