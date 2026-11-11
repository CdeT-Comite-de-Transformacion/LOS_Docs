package cmd

import (
	"bytes"
	"crypto/sha256"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CdeT-Comite-de-Transformacion/LOS_Docs/starter-kit/internal/config"
	"github.com/CdeT-Comite-de-Transformacion/LOS_Docs/starter-kit/internal/generator"
	"github.com/CdeT-Comite-de-Transformacion/LOS_Docs/starter-kit/internal/module"
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
	if err := gen.CreateModule(module.Options{Name: "sales", Type: "basic", CreateAll: template == "http"}); err != nil {
		t.Fatal(err)
	}

	// Audit is opt-in and inherited only by this module's repositories.
	if err := gen.AddPlatform("audit", "sales"); err != nil {
		t.Fatal(err)
	}
	if err := gen.CreateUseCase("sales", "submit-order", "", false); err != nil {
		t.Fatal(err)
	}
	if err := gen.CreateEntity("sales", "customer", []string{"name:string", "active:bool", "balance:int64"}, false); err != nil {
		t.Fatal(err)
	}

	// Multiple entities must share packages without overwriting each other.
	if err := gen.CreateEntity("sales", "order", nil, false); err != nil {
		t.Fatal(err)
	}
	if err := gen.CreateEntity("sales", "applicant", []string{"name:string"}, false, true); err != nil {
		t.Fatal(err)
	}
	if err := gen.CreateModule(module.Options{Name: "underwriting", Type: "basic", NoCRUD: true, CreateAll: true}); err != nil {
		t.Fatal(err)
	}
	if err := gen.CreateEntity("underwriting", "decision", nil, false); err != nil {
		t.Fatal(err)
	}
	if err := gen.ValidateModule("underwriting"); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		"internal/modules/sales/entities",
		"internal/modules/sales/application/applicant_service.go",
		"internal/modules/underwriting/application",
		"internal/modules/underwriting/http",
		"cmd/underwriting_routes.go",
	} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("unexpected generated path: %s (%v)", path, err)
		}
	}
	for _, path := range []string{
		"internal/modules/sales/domain/customer.go",
		"internal/modules/sales/domain/order.go",
		"internal/modules/sales/domain/applicant.go",
		"internal/modules/sales/application/customer_service.go",
		"internal/modules/sales/application/order_service.go",
		"internal/modules/underwriting/domain/decision.go",
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatal(err)
		}
	}

	if err := gen.CreateUseCase("sales", "approve-order", "order", false); err != nil {
		t.Fatal(err)
	}
	if err := gen.CreateUseCase("underwriting", "record-decision", "", false); err != nil {
		t.Fatal(err)
	}
	if err := gen.CreateUseCase("sales", "approve-order", "order", false); err == nil {
		t.Fatal("duplicate use case accepted")
	}
	if err := gen.CreateUseCase("sales", "invalid", "missing", false); err == nil {
		t.Fatal("missing repository accepted")
	}
	// Adding entities preserves custom composition and edited services.
	custom := []byte("package sales\n// custom composition\n")
	if err := os.WriteFile("internal/modules/sales/module.go", custom, 0644); err != nil {
		t.Fatal(err)
	}
	servicePath := "internal/modules/sales/application/customer_service.go"
	service, err := os.ReadFile(servicePath)
	if err != nil {
		t.Fatal(err)
	}
	service = append(service, []byte("\n// custom service\n")...)
	if err := os.WriteFile(servicePath, service, 0644); err != nil {
		t.Fatal(err)
	}
	if err := gen.CreateEntity("sales", "note", nil, false, true); err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string][]byte{"internal/modules/sales/module.go": custom, servicePath: service} {
		got, err := os.ReadFile(path)
		if err != nil || string(got) != string(want) {
			t.Fatalf("modified user file %s", path)
		}
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

func TestGeneratedConsoleBuild(t *testing.T) {
	if os.Getenv("MODKIT_TEST_POSTGRES_DSN") == "" {
		t.Skip("enable integration checks with MODKIT_TEST_POSTGRES_DSN")
	}
	dir := scaffold(t, "console")
	for _, args := range [][]string{{"mod", "tidy"}, {"test", "-buildvcs=false", "./..."}} {
		command := exec.Command("go", args...)
		command.Dir = dir
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("go %v: %v\n%s", args, err, output)
		}
	}
}

func snapshot(t *testing.T, root string) map[string][32]byte {
	t.Helper()
	result := map[string][32]byte{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			result[path] = [32]byte{}
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		result[path] = sha256.Sum256(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}
func TestDryRunDoesNotWrite(t *testing.T) {
	dir := scaffold(t, "http")
	before := snapshot(t, dir)
	cfg := config.Default()
	cfg.Modules.Root = "internal/modules"
	gen, err := generator.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	gen.DryRun = true
	var output bytes.Buffer
	gen.Output = &output
	for _, operation := range []func() error{
		func() error { return gen.CreateModule(module.Options{Name: "preview"}) },
		func() error { return gen.CreateEntity("sales", "preview", nil, false) },
		func() error { return gen.CreateUseCase("sales", "preview", "order", false) },
		func() error { return gen.AddPlatform("redis", "sales") },
		func() error {
			return gen.AddModuleElements(module.Options{Name: "sales", Elements: []string{"outbox"}, Force: true})
		},
	} {
		if err := operation(); err != nil {
			t.Fatal(err)
		}
	}
	after := snapshot(t, dir)
	if len(before) != len(after) {
		t.Fatal("dry run changed file count")
	}
	for path, hash := range before {
		if after[path] != hash {
			t.Fatalf("dry run changed %s", path)
		}
	}
	if !strings.Contains(output.String(), "CREATE") || !strings.Contains(output.String(), "UPDATE") {
		t.Fatalf("incomplete plan: %s", output.String())
	}
}

func TestOptionalAdapterGeneration(t *testing.T) {
	dir := scaffold(t, "http")
	cfg := config.Default()
	cfg.Modules.Root = "internal/modules"
	gen, err := generator.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, capability := range []string{"redis", "sns", "sqs", "minio"} {
		if err := gen.AddPlatform(capability, "sales"); err != nil {
			t.Fatal(err)
		}
	}
	// Reinstalling a component must retain manual adapter changes.
	path := "internal/platform/cache/redis.go"
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	before = append(before, []byte("\n// customized\n")...)
	if err := os.WriteFile(path, before, 0644); err != nil {
		t.Fatal(err)
	}
	if err := gen.AddPlatform("redis", "sales"); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(path)
	if err != nil || string(before) != string(after) {
		t.Fatal("adapter overwritten")
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
}

func TestAdapterIntegrations(t *testing.T) {
	if os.Getenv("MODKIT_TEST_ADAPTERS") != "1" {
		t.Skip("set MODKIT_TEST_ADAPTERS=1 for local protocol tests")
	}
	fixtures := map[string][]byte{}
	for path, name := range map[string]string{
		"internal/platform/cache/redis_test.go":        "redis_test.go.txt",
		"internal/platform/messaging/adapters_test.go": "messaging_test.go.txt",
		"internal/platform/storage/s3_test.go":         "storage_test.go.txt",
	} {
		data, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		fixtures[path] = data
	}
	dir := scaffold(t, "http")
	cfg := config.Default()
	cfg.Modules.Root = "internal/modules"
	gen, err := generator.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, capability := range []string{"redis", "sns", "sqs", "s3"} {
		if err := gen.AddPlatform(capability, "sales"); err != nil {
			t.Fatal(err)
		}
	}
	for path, data := range fixtures {
		if err := os.WriteFile(path, data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{
		{"mod", "edit", "-require=github.com/alicebob/miniredis/v2@v2.34.0"},
		{"mod", "tidy"},
		{"test", "-buildvcs=false", "./..."},
	} {
		command := exec.Command("go", args...)
		command.Dir = dir
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("go %v: %v\n%s", args, err, output)
		}
	}
}

func TestEnableAuditAfterEntityGeneration(t *testing.T) {
	dir := scaffold(t, "http")
	cfg := config.Default()
	cfg.Modules.Root = "internal/modules"
	gen, err := generator.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := gen.CreateModule(module.Options{Name: "late-audit"}); err != nil {
		t.Fatal(err)
	}
	if err := gen.CreateEntity("late-audit", "record", nil, false); err != nil {
		t.Fatal(err)
	}
	if err := gen.AddPlatform("audit", "late-audit"); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile("internal/modules/late-audit/infrastructure/record_repository.go")
	if err != nil || !strings.Contains(string(content), "audit.Record") {
		t.Fatal("existing repository was not wired to audit")
	}
	if err := gen.CreateModule(module.Options{Name: "custom-audit"}); err != nil {
		t.Fatal(err)
	}
	if err := gen.CreateEntity("custom-audit", "record", nil, false); err != nil {
		t.Fatal(err)
	}
	path := "internal/modules/custom-audit/infrastructure/record_repository.go"
	content, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(content, []byte("// custom")...), 0644); err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, dir)
	if err := gen.AddPlatform("audit", "custom-audit"); err == nil {
		t.Fatal("custom repository overwritten")
	}
	after := snapshot(t, dir)
	if len(before) != len(after) {
		t.Fatal("failed audit activation wrote files")
	}
	for path, hash := range before {
		if after[path] != hash {
			t.Fatalf("failed audit activation modified %s", path)
		}
	}
}

// Opt-in because generated projects may download their dependencies.
func TestGeneratedUnitTests(t *testing.T) {
	if os.Getenv("MODKIT_TEST_GENERATED") != "1" {
		t.Skip("set MODKIT_TEST_GENERATED=1")
	}
	for _, template := range []string{"http", "console"} {
		t.Run(template, func(t *testing.T) {
			dir := scaffold(t, template)
			if _, err := os.Stat(filepath.Join(dir, ".env.example")); err != nil {
				t.Fatal(err)
			}
			for _, args := range [][]string{{"mod", "tidy"}, {"test", "./..."}, {"vet", "./..."}} {
				command := exec.Command("go", args...)
				command.Dir = dir
				if output, err := command.CombinedOutput(); err != nil {
					t.Fatalf("go %v: %v\n%s", args, err, output)
				}
			}
		})
	}
}

func TestForcePreservesGeneratedTests(t *testing.T) {
	dir := scaffold(t, "http")
	cfg := config.Default()
	cfg.Modules.Root = "internal/modules"
	gen, err := generator.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	paths := []string{"internal/modules/sales/domain/customer_test.go", "internal/modules/sales/application/submit-order_usecase_test.go"}
	before := map[string]string{}
	for _, path := range paths {
		full := filepath.Join(dir, path)
		data, err := os.ReadFile(full)
		if err != nil {
			t.Fatal(err)
		}
		before[path] = string(data) + "\n// customized test\n"
		if err := os.WriteFile(full, []byte(before[path]), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := gen.CreateEntity("sales", "customer", []string{"name:string", "active:bool", "balance:int64"}, true); err != nil {
		t.Fatal(err)
	}
	if err := gen.CreateUseCase("sales", "submit-order", "", true); err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		data, err := os.ReadFile(filepath.Join(dir, path))
		if err != nil || string(data) != before[path] {
			t.Fatalf("customized tests changed: %s", path)
		}
	}
}

func TestModuleDocumentationIsPreserved(t *testing.T) {
	dir := scaffold(t, "http")
	readme := filepath.Join(dir, "internal/modules/sales/README.md")
	data, err := os.ReadFile(readme)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte("TODO(doc)")) {
		t.Fatal("missing documentation prompts")
	}
	customized := append(data, []byte("\nTeam-owned module decisions.\n")...)
	if err := os.WriteFile(readme, customized, 0644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.Modules.Root = "internal/modules"
	gen, err := generator.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := gen.CreateEntity("sales", "documented", nil, false, true); err != nil {
		t.Fatal(err)
	}
	if err := gen.CreateUseCase("sales", "document-operation", "", false); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(readme)
	if err != nil || !bytes.Equal(after, customized) {
		t.Fatal("module documentation overwritten")
	}
	if _, err := os.Stat(filepath.Join(dir, "docs/code-documentation.md")); err != nil {
		t.Fatal(err)
	}
}
