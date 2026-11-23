package generator

import (
	"bytes"
	"io/fs"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/CdeT-Comite-de-Transformacion/LOS_Docs/starter-kit/internal/config"
	templatefs "github.com/CdeT-Comite-de-Transformacion/LOS_Docs/starter-kit/internal/template"
)

func allPlatformFiles(cfg config.Config) (map[string][]byte, error) {
	files := map[string][]byte{}
	const base = "templates/platform/"
	err := fs.WalkDir(templatefs.FS(), "templates/platform", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		raw, err := templatefs.FS().ReadFile(path)
		if err != nil {
			return err
		}
		tmpl, err := template.New(path).Parse(string(raw))
		if err != nil {
			return err
		}
		var content bytes.Buffer
		if err := tmpl.Execute(&content, cfg.Project); err != nil {
			return err
		}
		output := strings.TrimSuffix(strings.TrimPrefix(path, base), ".tmpl")
		rendered := content.Bytes()
		if strings.HasSuffix(output, ".go") {
			rendered, err = instrument(rendered, cfg.Project.Module, output)
			if err != nil {
				return err
			}
		}
		files[filepath.FromSlash(output)] = rendered
		return nil
	})
	return files, err
}

func PlatformFiles(cfg config.Config) (map[string][]byte, error) {
	all, err := allPlatformFiles(cfg)
	if err != nil {
		return nil, err
	}
	files := map[string][]byte{}
	for path, content := range all {
		slash := filepath.ToSlash(path)
		if (strings.HasPrefix(slash, "internal/platform/") && !strings.HasPrefix(slash, "internal/platform/telemetry/")) || strings.Contains(slash, "platform_audit.sql") || (cfg.Project.Template == "console" && (slash == "internal/platform/telemetry/http.go" || slash == "internal/platform/telemetry/http_test.go")) {
			continue
		}
		files[path] = content
	}
	return files, nil
}
