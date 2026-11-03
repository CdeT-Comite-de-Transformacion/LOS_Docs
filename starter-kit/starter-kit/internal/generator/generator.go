package generator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/format"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"text/template"

	"github.com/your-org/modkit/internal/config"
	"github.com/your-org/modkit/internal/migration"
	"github.com/your-org/modkit/internal/module"
	"github.com/your-org/modkit/internal/naming"
	"github.com/your-org/modkit/internal/scaffold"
	templatefs "github.com/your-org/modkit/internal/template"
)

type Generator struct {
	cfg    config.Config
	DryRun bool
	Output io.Writer
}
type Field struct{ Name, Type, JSON string }
type templateData struct {
	ImportPath, RoutePath, TableName, Name, Package, ModuleDir, EntityName, ModulePath string
	Fields                                                                             []Field
	CRUD                                                                               bool
	Audit                                                                              bool
}
type entityState struct {
	Name   string
	Fields []Field
	CRUD   bool
}
type moduleState struct {
	Audit        bool
	Capabilities []string
	Version      int
	NoCRUD       bool
	HTTP         bool
	Entities     []entityState
	Files        []string
}

const metadataFile = ".modkit.json"

var namePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9]*([_-][a-zA-Z0-9]+)*$`)

func validName(name string) bool {
	return namePattern.MatchString(name) && token.IsIdentifier(naming.PackageName(name))
}
func New(cfg config.Config) (*Generator, error) {
	if cfg.Modules.Root == "" || filepath.IsAbs(cfg.Modules.Root) || filepath.Clean(cfg.Modules.Root) == ".." || strings.HasPrefix(filepath.Clean(cfg.Modules.Root), ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("modules.root must be a relative project directory")
	}
	return &Generator{cfg: cfg}, nil
}
func (g *Generator) destination(name string) string {
	return filepath.Join(g.cfg.Modules.Root, naming.DirectoryName(name))
}
func (g *Generator) data(moduleName string, entity entityState) templateData {
	dir := g.destination(moduleName)
	route := naming.DirectoryName(moduleName) + "/" + naming.DirectoryName(entity.Name)
	data := templateData{
		ImportPath: strings.TrimRight(g.cfg.Project.Module, "/") + "/" + filepath.ToSlash(dir),
		RoutePath:  route, TableName: strings.NewReplacer("/", "__", "-", "_").Replace(route),
		Name: entity.Name, Package: naming.PackageName(moduleName), ModuleDir: naming.DirectoryName(moduleName),
		EntityName: naming.Pascal(entity.Name), ModulePath: g.cfg.Project.Module, Fields: entity.Fields, CRUD: entity.CRUD,
	}
	if state, err := g.load(moduleName); err == nil {
		data.Audit = state.Audit
	}
	return data
}
func render(path string, data templateData) ([]byte, error) {
	raw, err := templatefs.FS().ReadFile("templates/module/basic/" + path + ".tmpl")
	if err != nil {
		return nil, err
	}
	tmpl, err := template.New(path).Parse(string(raw))
	if err != nil {
		return nil, err
	}
	var output bytes.Buffer
	if err := tmpl.Execute(&output, data); err != nil {
		return nil, err
	}
	if !strings.HasSuffix(path, ".go") {
		return output.Bytes(), nil
	}
	result, err := instrument(output.Bytes(), data.ModulePath, data.ModuleDir+"/"+path)
	if err != nil {
		return nil, fmt.Errorf("format %s: %w", path, err)
	}
	return result, nil
}
func putTemplate(files map[string][]byte, dest, output, source string, data templateData) error {
	content, err := render(source, data)
	if err != nil {
		return err
	}
	files[filepath.Join(dest, output)] = content
	return nil
}
func (g *Generator) load(name string) (moduleState, error) {
	var state moduleState
	if !validName(name) {
		return state, fmt.Errorf("invalid module name %q", name)
	}
	raw, err := os.ReadFile(filepath.Join(g.destination(name), metadataFile))
	if err != nil {
		return state, fmt.Errorf("module %q requires the shared-layer scaffold (.modkit.json); create a new module or migrate the existing one: %w", name, err)
	}
	if err = json.Unmarshal(raw, &state); err != nil {
		return state, err
	}
	if state.Version != 2 {
		return state, fmt.Errorf("unsupported module scaffold version; migrate pre-SQL scaffolds explicitly")
	}
	return state, nil
}
func (g *Generator) ValidateModule(name string) error {
	state, err := g.load(name)
	if err != nil {
		return err
	}
	for _, path := range append([]string{"module.go", "module_generated.go"}, state.Files...) {
		if _, err := os.Stat(filepath.Join(g.destination(name), path)); err != nil {
			return fmt.Errorf("module %q missing %s: %w", name, path, err)
		}
	}
	return nil
}
func (g *Generator) CreateModule(opts module.Options) error {
	if !validName(opts.Name) {
		return fmt.Errorf("invalid module name %q", opts.Name)
	}
	if opts.Type != "" && opts.Type != "basic" && opts.Type != "console" {
		return fmt.Errorf("unknown module template %q", opts.Type)
	}
	dest := g.destination(opts.Name)
	if _, err := os.Stat(dest); err == nil {
		return fmt.Errorf("module %q already exists; add entities or elements instead of recreating it", opts.Name)
	} else if !os.IsNotExist(err) {
		return err
	}
	state := moduleState{Version: 2, NoCRUD: opts.NoCRUD, HTTP: g.cfg.Project.Template == "http", Entities: []entityState{}}
	files := map[string][]byte{}
	files[filepath.Join(dest, "module.go")] = []byte("package " + naming.PackageName(opts.Name) + "\n\n// Add module-specific composition and integration helpers here.\n")
	data := g.data(opts.Name, entityState{})
	for _, path := range []string{"README.md", "domain/errors.go", "domain/events.go", "domain/events_test.go"} {
		if err := putTemplate(files, dest, path, path, data); err != nil {
			return err
		}
		state.Files = append(state.Files, path)
	}
	if err := g.addElements(opts, &state, files); err != nil {
		return err
	}
	return g.finish(opts.Name, state, files, nil)
}

var elementFiles = map[string][]string{
	"messaging":     {"infrastructure/event_publisher.go", "interfaces/consumer/handler.go"},
	"observability": {"interfaces/observability/hooks.go"},
	"outbox":        {"infrastructure/outbox.go"},
	"threading":     {"infrastructure/worker_group.go"},
}

func (g *Generator) addElements(opts module.Options, state *moduleState, files map[string][]byte) error {
	elements := append([]string{}, opts.Elements...)
	if opts.CreateAll {
		elements = append(elements, "messaging", "observability", "outbox", "threading")
		if !state.NoCRUD {
			elements = append(elements, "http")
		}
	}
	for _, element := range elements {
		if element == "http" {
			if state.NoCRUD {
				return fmt.Errorf("HTTP CRUD is disabled for module %q (--no-crud)", opts.Name)
			}
			state.HTTP = true
			continue
		}
		sources, ok := elementFiles[element]
		if !ok {
			return fmt.Errorf("unknown element %q", element)
		}
		for _, source := range sources {
			output := strings.TrimPrefix(source, "interfaces/")
			if err := putTemplate(files, g.destination(opts.Name), output, source, g.data(opts.Name, entityState{})); err != nil {
				return err
			}
			state.Files = appendUnique(state.Files, output)
		}
	}
	return nil
}
func (g *Generator) AddModuleElements(opts module.Options) error {
	state, err := g.load(opts.Name)
	if err != nil {
		return err
	}
	files := map[string][]byte{}
	if err := g.addElements(opts, &state, files); err != nil {
		return err
	}
	for _, entity := range state.Entities {
		if entity.CRUD && state.HTTP {
			testPath := "http/" + naming.DirectoryName(entity.Name) + "_handler_test.go"
			if _, err := os.Stat(filepath.Join(g.destination(opts.Name), testPath)); os.IsNotExist(err) {
				if err := putTemplate(files, g.destination(opts.Name), testPath, "interfaces/http/handler_test.go", g.data(opts.Name, entity)); err != nil {
					return err
				}
				state.Files = appendUnique(state.Files, testPath)
			}
			path := "http/" + naming.DirectoryName(entity.Name) + "_handler.go"
			if _, err := os.Stat(filepath.Join(g.destination(opts.Name), path)); os.IsNotExist(err) {
				if err := putTemplate(files, g.destination(opts.Name), path, "interfaces/http/handler.go", g.data(opts.Name, entity)); err != nil {
					return err
				}
				state.Files = appendUnique(state.Files, path)
			}
		}
	}
	overwrite := map[string]bool{}
	if opts.Force {
		for path := range files {
			overwrite[path] = true
		}
	}
	return g.finish(opts.Name, state, files, overwrite)
}
func appendUnique(items []string, item string) []string {
	for _, existing := range items {
		if existing == item {
			return items
		}
	}
	return append(items, item)
}

// CreateEntity adds named files to the module's shared layers.
func (g *Generator) CreateEntity(moduleName, name string, specs []string, force bool, noCRUD ...bool) error {
	if !validName(name) {
		return fmt.Errorf("invalid entity name %q", name)
	}
	state, err := g.load(moduleName)
	if err != nil {
		return err
	}
	if err := g.ValidateModule(moduleName); err != nil {
		return err
	}
	entity := entityState{Name: name, CRUD: !state.NoCRUD}
	if len(noCRUD) > 0 && noCRUD[0] {
		entity.CRUD = false
	}
	seen := map[string]bool{"Id": true, "Record": true, "PullEvents": true, "TableName": true}
	allowed := map[string]bool{"string": true, "bool": true, "int": true, "int64": true, "uint": true, "uint64": true, "float64": true}
	for _, spec := range specs {
		parts := strings.Split(spec, ":")
		if len(parts) != 2 || !validName(parts[0]) || !allowed[parts[1]] {
			return fmt.Errorf("invalid field %q; use name:string|bool|int|int64|uint|uint64|float64", spec)
		}
		field := naming.Pascal(parts[0])
		if seen[field] {
			return fmt.Errorf("duplicate or reserved field %q", parts[0])
		}
		seen[field] = true
		entity.Fields = append(entity.Fields, Field{field, parts[1], strings.ReplaceAll(naming.DirectoryName(parts[0]), "-", "_")})
	}
	// Check generated domain symbols before writing anything.
	symbols := func(e entityState) []string {
		n := naming.Pascal(e.Name)
		out := []string{n, "New" + n, n + "Created", "New" + n + "Created"}
		if e.CRUD {
			out = append(out, n+"Repository", n+"Page")
		}
		return out
	}
	reserved := map[string]bool{}
	for _, n := range []string{"Event", "BaseEvent", "NewBaseEvent", "Pagination", "DefaultLimit", "MaxLimit", "ErrInvalidID", "ErrNotFound", "ErrConflict", "ErrInvalidPagination"} {
		reserved[n] = true
	}
	index := -1
	for i, existing := range state.Entities {
		if naming.DirectoryName(existing.Name) == naming.DirectoryName(name) {
			index = i
			if !force {
				return fmt.Errorf("entity %q already exists", name)
			}
			if existing.CRUD != entity.CRUD {
				return fmt.Errorf("--force cannot change CRUD mode; migrate existing files explicitly")
			}
		} else {
			for _, symbol := range symbols(existing) {
				reserved[symbol] = true
			}
		}
	}
	for _, symbol := range symbols(entity) {
		if reserved[symbol] {
			return fmt.Errorf("entity name conflicts with domain symbol %s", symbol)
		}
	}
	files := map[string][]byte{}
	dest := g.destination(moduleName)
	stem := naming.DirectoryName(name)
	mappings := map[string]string{"domain/" + stem + ".go": "domain/entity.go", "domain/" + stem + "_events.go": "domain/event.go"}
	mappings["domain/"+stem+"_test.go"] = "domain/entity_test.go"
	if entity.CRUD {
		mappings["application/"+stem+"_service_test.go"] = "application/service_test.go"
		mappings["domain/"+stem+"_repository.go"] = "domain/repository.go"
		mappings["application/"+stem+"_service.go"] = "application/service.go"
		mappings["infrastructure/"+stem+"_repository.go"] = "infrastructure/repository.go"
		mappings["infrastructure/"+stem+"_repository_test.go"] = "infrastructure/repository_test.go"
		if state.HTTP {
			mappings["http/"+stem+"_handler.go"] = "interfaces/http/handler.go"
			mappings["http/"+stem+"_handler_test.go"] = "interfaces/http/handler_test.go"
		}
		for _, shared := range []string{"domain/pagination.go", "application/ports.go"} {
			if _, err := os.Stat(filepath.Join(dest, shared)); os.IsNotExist(err) {
				mappings[shared] = shared
			}
		}
	}
	overwrite := map[string]bool{}
	for output, source := range mappings {
		if strings.HasSuffix(output, "_test.go") {
			if _, err := os.Stat(filepath.Join(dest, output)); err == nil {
				continue
			}
		}
		if err := putTemplate(files, dest, output, source, g.data(moduleName, entity)); err != nil {
			return err
		}
		state.Files = appendUnique(state.Files, output)
		if force {
			overwrite[filepath.Join(dest, output)] = true
		}
	}
	if entity.CRUD {
		if index >= 0 && !reflect.DeepEqual(state.Entities[index].Fields, entity.Fields) {
			return fmt.Errorf("CRUD schema changes require a new migration and explicit model update; --force never rewrites migration history")
		}
		if index < 0 {
			path, err := migration.NextPath("create_" + g.data(moduleName, entity).TableName)
			if err != nil {
				return err
			}
			files[path] = entityMigration(g.data(moduleName, entity))
		}
	}
	if index < 0 {
		state.Entities = append(state.Entities, entity)
	} else {
		state.Entities[index] = entity
	}
	return g.finish(moduleName, state, files, overwrite)
}
func (g *Generator) finish(name string, state moduleState, files map[string][]byte, overwrite map[string]bool) error {
	if overwrite == nil {
		overwrite = map[string]bool{}
	}
	dest := g.destination(name)
	data := g.data(name, entityState{})
	crud := []entityState{}
	for _, entity := range state.Entities {
		if entity.CRUD {
			crud = append(crud, entity)
		}
	}
	var source strings.Builder
	fmt.Fprintf(&source, "// Code generated by modkit. DO NOT EDIT. Customize module.go instead.\npackage %s\n", data.Package)
	if len(crud) == 0 {
		source.WriteString("type Module struct{}\nfunc New() *Module { return &Module{} }\n")
	} else {
		fmt.Fprintf(&source, "import (\n\"gorm.io/gorm\"\n%q\n%q\n", data.ImportPath+"/application", data.ImportPath+"/infrastructure")
		if state.HTTP {
			fmt.Fprintf(&source, "\"github.com/gin-gonic/gin\"\nmodulehttp %q\n", data.ImportPath+"/http")
		}
		source.WriteString(")\ntype Module struct {\n")
		for _, e := range crud {
			n := naming.Pascal(e.Name)
			fmt.Fprintf(&source, "%s *application.%sService\n", n, n)
		}
		source.WriteString("}\nfunc New(db *gorm.DB) (*Module,error) {\nreturn &Module{\n")
		for _, e := range crud {
			n := naming.Pascal(e.Name)
			fmt.Fprintf(&source, "%s: application.New%sService(infrastructure.New%sRepository(db)),\n", n, n, n)
		}
		source.WriteString("},nil\n}\n")
		if state.HTTP {
			source.WriteString("func(m *Module) RegisterRoutes(router *gin.Engine){\n")
			for _, e := range crud {
				n := naming.Pascal(e.Name)
				fmt.Fprintf(&source, "modulehttp.New%sHandler(m.%s).RegisterRoutes(router)\n", n, n)
			}
			source.WriteString("}\n")
		}
	}
	formatted, err := format.Source([]byte(source.String()))
	if err != nil {
		return err
	}
	generated := filepath.Join(dest, "module_generated.go")
	files[generated] = formatted
	overwrite[generated] = true
	if state.HTTP && len(crud) > 0 && g.cfg.Project.Template == "http" {
		route := filepath.Join("cmd", naming.DirectoryName(name)+"_routes.go")
		content := fmt.Sprintf(`// Code generated by modkit. DO NOT EDIT.
package cmd
import ("github.com/gin-gonic/gin"; "gorm.io/gorm"; module %q)
func init(){
 moduleInitializers=append(moduleInitializers,func(db *gorm.DB,router *gin.Engine)error{
  m,err:=module.New(db);if err!=nil{return err}
  m.RegisterRoutes(router)
  return nil
 })
}
`, data.ImportPath)
		formatted, err := format.Source([]byte(content))
		if err != nil {
			return err
		}
		files[route] = formatted
		// Only replace registrations owned by this generator.
		if old, err := os.ReadFile(route); err == nil && strings.HasPrefix(string(old), "// Code generated by modkit.") {
			overwrite[route] = true
		}
	}
	raw, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	metadata := filepath.Join(dest, metadataFile)
	files[metadata] = append(raw, '\n')
	overwrite[metadata] = true
	return scaffold.Write(files, overwrite, g.DryRun, g.Output)
}

func entityMigration(data templateData) []byte {
	types := map[string]string{"string": "TEXT NOT NULL DEFAULT ''", "bool": "BOOLEAN NOT NULL DEFAULT FALSE", "int": "BIGINT NOT NULL DEFAULT 0", "int64": "BIGINT NOT NULL DEFAULT 0", "uint": "NUMERIC(20,0) NOT NULL DEFAULT 0", "uint64": "NUMERIC(20,0) NOT NULL DEFAULT 0", "float64": "DOUBLE PRECISION NOT NULL DEFAULT 0"}
	columns := []string{`"id" TEXT PRIMARY KEY`}
	for _, field := range data.Fields {
		columns = append(columns, fmt.Sprintf("%q %s", field.JSON, types[field.Type]))
	}
	return []byte(fmt.Sprintf("-- +goose Up\nCREATE TABLE %q (\n %s\n);\n\n-- +goose Down\nDROP TABLE %q;\n", data.TableName, strings.Join(columns, ",\n "), data.TableName))
}
