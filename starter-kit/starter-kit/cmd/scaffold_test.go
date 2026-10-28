package cmd

import (
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/your-org/modkit/internal/config"
	"github.com/your-org/modkit/internal/generator"
	"github.com/your-org/modkit/internal/module"
)

func scaffold(t *testing.T, template string) string {
	t.Helper()
	dir := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
	cfg := config.Default()
	cfg.Project.Template = template
	cfg.Modules.Root = "internal/modules"
	if err := initProject(cfg); err != nil {
		t.Fatal(err)
	}
	gen, err := generator.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := gen.CreateModule(module.Options{Name: "sales", Type: "basic", CreateAll: true}); err != nil {
		t.Fatal(err)
	}
	if err := gen.CreateEntity("sales", "customer", []string{"name:string", "active:bool", "balance:int64"}, false); err != nil {
		t.Fatal(err)
	}
	if err := gen.CreateEntity("sales", "customer", nil, false); err == nil {
		t.Fatal("duplicate entity accepted")
	}
	for _, name := range []string{"../escape", "bad/name", "type"} {
		if err := gen.CreateEntity("sales", name, nil, false); err == nil {
			t.Fatalf("invalid name accepted: %s", name)
		}
	}
	for _, field := range []string{"id:string", "name:unknown", "Record:string"} {
		if err := gen.CreateEntity("sales", "invalid", []string{field}, false); err == nil {
			t.Fatalf("invalid field accepted: %s", field)
		}
	}
	if err := gen.CreateEntity("missing", "customer", nil, false); err == nil {
		t.Fatal("missing module accepted")
	}
	if err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.HasSuffix(path, ".go") {
			_, err = parser.ParseFile(token.NewFileSet(), path, nil, parser.AllErrors)
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "gorm.io/driver/postgres") {
		t.Fatal("missing PostgreSQL dependency")
	}
	return dir
}

func TestGeneratedScaffolds(t *testing.T) {
	for _, template := range []string{"http", "console"} {
		t.Run(template, func(t *testing.T) { scaffold(t, template) })
	}
}

// Runs CRUD against an isolated schema rolled back at test completion.
func TestPostgresCRUD(t *testing.T) {
	dsn := os.Getenv("MODKIT_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("set MODKIT_TEST_POSTGRES_DSN to run generated PostgreSQL CRUD")
	}
	fixture, err := os.ReadFile("testdata/crud_test.go.txt")
	if err != nil {
		t.Fatal(err)
	}
	dir := scaffold(t, "http")
	if err := os.WriteFile(filepath.Join(dir, "cmd", "crud_test.go"), fixture, 0644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"mod", "tidy"}, {"test", "-buildvcs=false", "./..."}} {
		command := exec.Command("go", args...)
		command.Dir = dir
		command.Env = append(os.Environ(), "DB_DSN="+dsn)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("go %v: %v\n%s", args, err, output)
		}
	}
}
