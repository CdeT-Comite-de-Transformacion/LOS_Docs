package generator

import (
	"bytes"
	"fmt"
	"go/format"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	texttemplate "text/template"

	"github.com/your-org/modkit/internal/config"
	"github.com/your-org/modkit/internal/module"
	"github.com/your-org/modkit/internal/naming"
	templatefs "github.com/your-org/modkit/internal/template"
)

type Generator struct {
	cfg    config.Config
	parent string
	fields []Field
}

type templateData struct {
	ImportPath string
	RoutePath  string
	TableName  string
	Fields     []Field
	Name       string
	Package    string
	ModuleDir  string
	EntityName string
	ModulePath string
}

var coreFiles = map[string]bool{
	"module.go":                    true,
	"application/ports.go":         true,
	"application/service.go":       true,
	"domain/entity.go":             true,
	"domain/errors.go":             true,
	"domain/event.go":              true,
	"domain/repository.go":         true,
	"infrastructure/repository.go": true,
}

var elementFiles = map[string][]string{
	"http": {
		"interfaces/http/handler.go",
	},
	"messaging": {
		"infrastructure/event_publisher.go",
		"interfaces/consumer/handler.go",
	},
	"observability": {
		"interfaces/observability/hooks.go",
	},
	"outbox": {
		"infrastructure/outbox.go",
	},
	"threading": {
		"infrastructure/worker_group.go",
	},
}

func New(cfg config.Config) (*Generator, error) {
	if cfg.Modules.Root == "" || filepath.IsAbs(cfg.Modules.Root) || filepath.Clean(cfg.Modules.Root) == ".." || strings.HasPrefix(filepath.Clean(cfg.Modules.Root), ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("modules.root cannot be empty")
	}
	return &Generator{cfg: cfg}, nil
}

func (g *Generator) CreateModule(opts module.Options) error {
	if g.cfg.Project.Template == "http" {
		opts.Elements = append(opts.Elements, "http")
	}
	return g.generateModule(opts, false)
}

func (g *Generator) AddModuleElements(opts module.Options) error {
	return g.generateModule(opts, true)
}

func (g *Generator) ValidateModule(name string) error {
	moduleDir := naming.DirectoryName(name)
	if strings.TrimSpace(name) == "" || moduleDir == "" {
		return fmt.Errorf("invalid module name %q", name)
	}

	destination := filepath.Join(g.cfg.Modules.Root, moduleDir)
	missing := make([]string, 0)
	for path := range coreFiles {
		if _, err := os.Stat(filepath.Join(destination, filepath.FromSlash(path))); err != nil {
			if os.IsNotExist(err) {
				missing = append(missing, path)
				continue
			}
			return err
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("module %q is missing: %s", name, strings.Join(missing, ", "))
	}
	return nil
}

func (g *Generator) generateModule(opts module.Options, existing bool) error {
	if !validName(opts.Name) {
		return fmt.Errorf("module name is required")
	}

	packageName := naming.PackageName(opts.Name)
	moduleDir := naming.DirectoryName(opts.Name)
	if packageName == "" || moduleDir == "" {
		return fmt.Errorf("invalid module name %q", opts.Name)
	}

	templateType := opts.Type
	if templateType == "console" || templateType == "basic" {
		templateType = "basic"
	}
	base := strings.Join([]string{"templates", "module", templateType}, "/")
	if _, err := fs.Stat(templatefs.FS(), base+"/module.go.tmpl"); err != nil {
		return fmt.Errorf("template %q not found", opts.Type)
	}

	destination := filepath.Join(g.cfg.Modules.Root, moduleDir)
	if _, err := os.Stat(destination); err == nil && !existing && !opts.Force {
		return fmt.Errorf("module %q already exists; use --force to overwrite generated files", moduleDir)
	}
	if _, err := os.Stat(destination); os.IsNotExist(err) && existing {
		return fmt.Errorf("module %q does not exist; create it first", moduleDir)
	}

	data := templateData{
		Name:       opts.Name,
		Package:    packageName,
		ModuleDir:  moduleDir,
		EntityName: naming.Pascal(opts.Name),
		ModulePath: g.cfg.Project.Module,
		ImportPath: strings.TrimRight(g.cfg.Project.Module, "/") + "/" + filepath.ToSlash(destination),
		RoutePath:  strings.TrimPrefix(g.parent+"/"+moduleDir, "/"),
		TableName:  strings.NewReplacer("/", "__", "-", "_").Replace(strings.TrimPrefix(g.parent+"/"+moduleDir, "/")),
		Fields:     g.fields,
	}

	err := fs.WalkDir(templatefs.FS(), base, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		rel := strings.TrimPrefix(path, base+"/")

		outputRel := strings.TrimSuffix(rel, ".tmpl")
		if !shouldCreate(outputRel, opts, existing) {
			return nil
		}
		outputPath := filepath.Join(destination, filepath.FromSlash(outputRel))
		if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
			return err
		}

		content, err := fs.ReadFile(templatefs.FS(), path)
		if err != nil {
			return err
		}

		tmpl, err := texttemplate.New(path).Parse(string(content))
		if err != nil {
			return err
		}

		var rendered bytes.Buffer
		if err := tmpl.Execute(&rendered, data); err != nil {
			return err
		}
		formatted, err := format.Source(rendered.Bytes())
		if err != nil {
			return fmt.Errorf("format %s: %w", path, err)
		}

		flags := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
		if !opts.Force {
			flags |= os.O_EXCL
		}
		file, err := os.OpenFile(outputPath, flags, 0o644)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = file.Write(formatted)
		return err
	})
	if err != nil {
		return err
	}
	if g.cfg.Project.Template == "http" && shouldCreateHTTP(opts) {
		return g.registerHTTPRoutes(data, opts.Force)
	}
	return nil
}

func shouldCreateHTTP(opts module.Options) bool {
	if opts.CreateAll {
		return true
	}
	for _, element := range opts.Elements {
		if element == "http" {
			return true
		}
	}
	return false
}

func (g *Generator) registerHTTPRoutes(data templateData, force bool) error {
	if err := os.MkdirAll("cmd", 0o755); err != nil {
		return err
	}
	path := filepath.Join("cmd", strings.ReplaceAll(data.RoutePath, "/", "_")+"_routes.go")
	content := fmt.Sprintf(`package cmd

import (
 "github.com/gin-gonic/gin"
 "gorm.io/gorm"
 entitymodule "%s"
 entityhttp "%s/interfaces/http"
)

func init() {
 moduleInitializers = append(moduleInitializers, func(db *gorm.DB, router *gin.Engine) error {
  module, err := entitymodule.New(db)
  if err != nil { return err }
  entityhttp.NewHandler(module.Service).RegisterRoutes(router)
  return nil
 })
}
`, data.ImportPath, data.ImportPath)

	formatted, err := format.Source([]byte(content))
	if err != nil {
		return err
	}
	flags := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	if !force {
		flags |= os.O_EXCL
	}
	file, err := os.OpenFile(path, flags, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(formatted)
	return err
}

func shouldCreate(path string, opts module.Options, existing bool) bool {
	if opts.CreateAll || (!existing && coreFiles[path]) {
		return true
	}
	for _, element := range opts.Elements {
		for _, elementPath := range elementFiles[element] {
			if path == elementPath {
				return true
			}
		}
	}
	return false
}

type Field struct{ Name, Type, JSON string }

var namePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9]*([_-][a-zA-Z0-9]+)*$`)

func validName(name string) bool {
	return namePattern.MatchString(name) && token.IsIdentifier(naming.PackageName(name))
}

// CreateEntity adds an isolated aggregate inside an existing module.
func (g *Generator) CreateEntity(moduleName, name string, specs []string, force bool) error {
	if !validName(moduleName) || !validName(name) {
		return fmt.Errorf("invalid module or entity name")
	}
	if err := g.ValidateModule(moduleName); err != nil {
		return err
	}
	fields := make([]Field, 0, len(specs))
	seen := map[string]bool{"ID": true, "TableName": true, "Record": true, "PullEvents": true}
	allowed := map[string]bool{"string": true, "bool": true, "int": true, "int64": true, "uint": true, "uint64": true, "float64": true}
	for _, spec := range specs {
		parts := strings.Split(spec, ":")
		if len(parts) != 2 || !validName(parts[0]) || !allowed[parts[1]] {
			return fmt.Errorf("invalid field %q; use name:string|bool|int|int64|uint|uint64|float64", spec)
		}
		fieldName := naming.Pascal(parts[0])
		if strings.EqualFold(fieldName, "id") || seen[fieldName] {
			return fmt.Errorf("duplicate or reserved field %q", parts[0])
		}
		seen[fieldName] = true
		fields = append(fields, Field{Name: fieldName, Type: parts[1], JSON: strings.ReplaceAll(naming.DirectoryName(parts[0]), "-", "_")})
	}
	child := *g
	child.parent = naming.DirectoryName(moduleName)
	child.fields = fields
	child.cfg.Modules.Root = filepath.Join(g.cfg.Modules.Root, child.parent, "entities")
	return child.CreateModule(module.Options{Name: name, Type: g.cfg.Defaults.Template, Force: force})
}
