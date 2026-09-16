package generator

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	texttemplate "text/template"

	"github.com/CdeT-Comite-de-Transformacion/los-bff/starter-kit/internal/config"
	"github.com/CdeT-Comite-de-Transformacion/los-bff/starter-kit/internal/module"
	"github.com/CdeT-Comite-de-Transformacion/los-bff/starter-kit/internal/naming"
	templatefs "github.com/CdeT-Comite-de-Transformacion/los-bff/starter-kit/internal/template"
)

type Generator struct {
	cfg config.Config
}

type templateData struct {
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
	if cfg.Modules.Root == "" {
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
	if strings.TrimSpace(opts.Name) == "" {
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

		flags := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
		if !opts.Force {
			flags |= os.O_EXCL
		}
		file, err := os.OpenFile(outputPath, flags, 0o644)
		if err != nil {
			return err
		}
		defer file.Close()

		return tmpl.Execute(file, data)
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
	path := filepath.Join("cmd", data.Package+"_routes.go")
	content := fmt.Sprintf(`package cmd

import (
	%s "%s/internal/%s"
	http%s "%s/internal/%s/interfaces/http"
)

func init() {
	module := %s.New()
	http%s.NewHandler(module.Service).RegisterRoutes(router)
}
`, data.Package, data.ModulePath, data.ModuleDir, data.Package, data.ModulePath, data.ModuleDir, data.Package, data.Package)

	flags := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	if !force {
		flags |= os.O_EXCL
	}
	file, err := os.OpenFile(path, flags, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(content)
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
