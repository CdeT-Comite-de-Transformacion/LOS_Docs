package migration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func inProject(t *testing.T) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
	if err := os.MkdirAll(Directory, 0755); err != nil {
		t.Fatal(err)
	}
}
func TestConflictsAndMergePreserveSources(t *testing.T) {
	inProject(t)
	a := []byte("-- +goose Up\nCREATE TABLE a(id int);\n-- +goose Down\nDROP TABLE a;\n")
	b := []byte("-- +goose Up\nCREATE TABLE b(id int);\n-- +goose Down\nDROP TABLE b;\n")
	for name, data := range map[string][]byte{"10_a.sql": a, "10_b.sql": b} {
		if err := os.WriteFile(filepath.Join(Directory, name), data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := Check(); err == nil || !strings.Contains(err.Error(), "duplicate version") {
		t.Fatalf("missing conflict: %v", err)
	}
	merged, err := Merge([]string{"10_a.sql", "10_b.sql"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Index(string(merged), "CREATE TABLE a") > strings.Index(string(merged), "CREATE TABLE b") || strings.Index(string(merged), "DROP TABLE b") > strings.Index(string(merged), "DROP TABLE a") {
		t.Fatal("incorrect merge order")
	}
	for name, want := range map[string][]byte{"10_a.sql": a, "10_b.sql": b} {
		got, err := os.ReadFile(filepath.Join(Directory, name))
		if err != nil || string(got) != string(want) {
			t.Fatal("source changed")
		}
	}
	if _, err := Merge([]string{"../escape.sql", "10_a.sql"}); err == nil {
		t.Fatal("path traversal accepted")
	}
}
func TestCheckRejectsIncompleteAndConflictMarkers(t *testing.T) {
	inProject(t)
	path := filepath.Join(Directory, "1_test.sql")
	for _, content := range []string{string(Draft()), "-- +goose Up\n<<<<<<< branch\n-- +goose Down\nSELECT 1;", "-- +goose Up\nSELECT 1;"} {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		if err := Check(); err == nil {
			t.Fatal("invalid migration accepted")
		}
	}
	if err := os.WriteFile(path, []byte("-- +goose Up\nSELECT 1;\n-- +goose Down\nSELECT 1;"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := Check(); err != nil {
		t.Fatal(err)
	}
}
