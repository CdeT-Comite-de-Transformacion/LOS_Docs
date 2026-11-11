package generator

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"

	"github.com/CdeT-Comite-de-Transformacion/LOS_Docs/starter-kit/internal/migration"
	"github.com/CdeT-Comite-de-Transformacion/LOS_Docs/starter-kit/internal/naming"
	"github.com/CdeT-Comite-de-Transformacion/LOS_Docs/starter-kit/internal/scaffold"
	"golang.org/x/mod/modfile"
)

type capability struct {
	Sources                    []string
	Dependencies               map[string]string
	Package, Constructor, Type string
}

var capabilities = map[string]capability{
	"audit": {Sources: []string{"audit/audit.go"}},
	"redis": {Sources: []string{"cache/cache.go", "cache/redis.go"}, Dependencies: map[string]string{"github.com/redis/go-redis/v9": "v9.7.3"}, Package: "cache", Constructor: "OpenRedis", Type: "Redis"},
	"sns":   {Sources: []string{"messaging/messaging.go", "messaging/sns.go"}, Dependencies: map[string]string{"github.com/aws/aws-sdk-go-v2/config": "v1.29.9", "github.com/aws/aws-sdk-go-v2/service/sns": "v1.33.14"}, Package: "messaging", Constructor: "OpenSNS", Type: "SNS"},
	"sqs":   {Sources: []string{"messaging/messaging.go", "messaging/sqs.go"}, Dependencies: map[string]string{"github.com/aws/aws-sdk-go-v2/config": "v1.29.9", "github.com/aws/aws-sdk-go-v2/service/sqs": "v1.38.0"}, Package: "messaging", Constructor: "OpenSQS", Type: "SQS"},
	"s3":    {Sources: []string{"storage/storage.go", "storage/s3.go"}, Dependencies: map[string]string{"github.com/aws/aws-sdk-go-v2/config": "v1.29.9", "github.com/aws/aws-sdk-go-v2/service/s3": "v1.78.0"}, Package: "storage", Constructor: "OpenS3", Type: "S3"},
}

// AddPlatform installs adapters once and optionally generates explicit module wiring.
// Existing adapters are preserved. No connections or credential lookups happen here.
func (g *Generator) AddPlatform(name, moduleName string) error {
	if name == "minio" {
		name = "s3"
	}
	cap, ok := capabilities[name]
	if !ok {
		return fmt.Errorf("unknown component %q; use audit, redis, sns, sqs, s3 or minio", name)
	}
	var state moduleState
	var err error
	if moduleName != "" {
		state, err = g.load(moduleName)
		if err != nil {
			return err
		}
		if err := g.ValidateModule(moduleName); err != nil {
			return err
		}
	}
	templates, err := allPlatformFiles(g.cfg)
	if err != nil {
		return err
	}
	files := map[string][]byte{}
	overwrite := map[string]bool{}
	newAudit := false
	for _, source := range cap.Sources {
		path := filepath.Join("internal", "platform", source)
		if _, err := os.Stat(path); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return err
		}
		raw, ok := templates[path]
		if !ok {
			return fmt.Errorf("adapter template missing: %s", path)
		}
		formatted, err := format.Source(raw)
		if err != nil {
			return err
		}
		files[path] = formatted
		if name == "audit" {
			newAudit = true
		}
	}
	if newAudit {
		path, err := migration.NextPath("platform_audit")
		if err != nil {
			return err
		}
		files[path] = templates[filepath.Join("migrations", "sql", "00001_platform_audit.sql")]
	}
	if len(cap.Dependencies) > 0 {
		raw, err := os.ReadFile("go.mod")
		if err != nil {
			return err
		}
		mod, err := modfile.Parse("go.mod", raw, nil)
		if err != nil {
			return err
		}
		for dependency, version := range cap.Dependencies {
			present := false
			for _, require := range mod.Require {
				if require.Mod.Path == dependency {
					present = true
				}
			}
			if !present {
				if err := mod.AddRequire(dependency, version); err != nil {
					return err
				}
			}
		}
		mod.Cleanup()
		content, err := mod.Format()
		if err != nil {
			return err
		}
		files["go.mod"] = content
		overwrite["go.mod"] = true
	}
	if moduleName != "" {
		if name == "audit" && !state.Audit {
			for _, entity := range state.Entities {
				if !entity.CRUD {
					continue
				}
				data := g.data(moduleName, entity)
				path := filepath.Join(g.destination(moduleName), "infrastructure", naming.DirectoryName(entity.Name)+"_repository.go")
				current, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				original, err := render("infrastructure/repository.go", data)
				if err != nil {
					return err
				}
				if !bytes.Equal(current, original) {
					return fmt.Errorf("%s contains custom code; integrate audit.Record in its transaction manually before enabling module auditing", path)
				}
				data.Audit = true
				content, err := render("infrastructure/repository.go", data)
				if err != nil {
					return err
				}
				files[path] = content
				overwrite[path] = true
			}
			state.Audit = true
		} else if name != "audit" {
			relative := "infrastructure/" + name + "_adapter.go"
			path := filepath.Join(g.destination(moduleName), relative)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				source := fmt.Sprintf(`package infrastructure
import ("context"; %q)
// New%sAdapter is explicit: call once from module.go and reuse the client.
func New%sAdapter(ctx context.Context)(*%s.%s,error){return %s.%s(ctx)}
`, g.cfg.Project.Module+"/internal/platform/"+cap.Package, strings.ToUpper(name), strings.ToUpper(name), cap.Package, cap.Type, cap.Package, cap.Constructor)
				content, err := format.Source([]byte(source))
				if err != nil {
					return err
				}
				files[path] = content
			} else if err != nil {
				return err
			}
			state.Files = appendUnique(state.Files, relative)
		}
		state.Capabilities = appendUnique(state.Capabilities, name)
		return g.finish(moduleName, state, files, overwrite)
	}
	return scaffold.Write(files, overwrite, g.DryRun, g.Output)
}

type CapabilityStatus struct {
	Name               string
	Installed, Enabled bool
}

func (g *Generator) PlatformStatus(moduleName string) ([]CapabilityStatus, error) {
	enabled := map[string]bool{}
	if moduleName != "" {
		state, err := g.load(moduleName)
		if err != nil {
			return nil, err
		}
		for _, name := range state.Capabilities {
			enabled[name] = true
		}
	}
	result := []CapabilityStatus{}
	for _, name := range []string{"postgres", "audit", "redis", "sns", "sqs", "s3"} {
		paths := []string{"internal/platform/database/database.go"}
		if name != "postgres" {
			paths = nil
			for _, source := range capabilities[name].Sources {
				paths = append(paths, filepath.Join("internal", "platform", source))
			}
		}
		installed := true
		for _, path := range paths {
			if _, err := os.Stat(path); os.IsNotExist(err) {
				installed = false
			} else if err != nil {
				return nil, err
			}
		}
		result = append(result, CapabilityStatus{Name: name, Installed: installed, Enabled: enabled[name]})
	}
	return result, nil
}
