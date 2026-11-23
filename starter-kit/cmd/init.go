package cmd

import (
	"bufio"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/CdeT-Comite-de-Transformacion/LOS_Docs/starter-kit/internal/config"
	"github.com/CdeT-Comite-de-Transformacion/LOS_Docs/starter-kit/internal/generator"
	"github.com/CdeT-Comite-de-Transformacion/LOS_Docs/starter-kit/internal/naming"
	fileplan "github.com/CdeT-Comite-de-Transformacion/LOS_Docs/starter-kit/internal/scaffold"
	"gopkg.in/yaml.v3"
)

func initProject(cfg config.Config) error {
	return initProjectWithTemplate(cfg, cfg.Project.Template)
}

func initProjectWithTemplate(cfg config.Config, projectTemplate string) error {
	projectModule := strings.TrimSpace(cfg.Project.Module)
	if projectModule == "" {
		return fmt.Errorf("project.module cannot be empty")
	}
	if projectTemplate == "" {
		projectTemplate = "http"
	}
	if projectTemplate != "console" && projectTemplate != "http" {
		return fmt.Errorf("unknown project template %q; use http or console", projectTemplate)
	}

	files, err := projectFiles(cfg, projectTemplate)
	if err != nil {
		return err
	}
	return fileplan.Write(files, nil, false, nil)
}

func projectFiles(cfg config.Config, projectTemplate string) (map[string][]byte, error) {
	projectModule := cfg.Project.Module
	if projectTemplate == "" {
		projectTemplate = "http"
	}
	if projectTemplate != "http" && projectTemplate != "console" {
		return nil, fmt.Errorf("unknown project template %q", projectTemplate)
	}
	files := map[string][]byte{}
	files["main.go"] = []byte(fmt.Sprintf("package main\nimport %q\nfunc main(){cmd.Execute()}\n", projectModule+"/cmd"))
	files[filepath.Join("cmd", "root.go")] = []byte(projectRootContent(cfg.Project.Name, projectTemplate, projectModule))
	files[filepath.Join("internal", "platform", "database", "database.go")] = []byte(strings.ReplaceAll(databaseContent, "PROJECT_MODULE", projectModule))
	goMod := fmt.Sprintf("module %s\n\ngo 1.24\n\nrequire (\n gorm.io/gorm v1.30.0\n github.com/DATA-DOG/go-sqlmock v1.5.2\n gorm.io/driver/postgres v1.6.0\n github.com/pressly/goose/v3 v3.24.3\n go.opentelemetry.io/otel v1.38.0\n go.opentelemetry.io/otel/trace v1.38.0\n go.opentelemetry.io/otel/sdk v1.38.0\n go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.38.0\n)\n", projectModule)
	if projectTemplate == "http" {
		goMod += "\nrequire github.com/gin-gonic/gin v1.11.0\n"
	}
	files["go.mod"] = []byte(goMod)
	platform, err := generator.PlatformFiles(cfg)
	if err != nil {
		return nil, err
	}
	for path, content := range platform {
		files[path] = content
	}
	for path, content := range files {
		if strings.HasSuffix(path, ".go") {
			formatted, err := format.Source(content)
			if err != nil {
				return nil, fmt.Errorf("format %s: %w", path, err)
			}
			files[path] = formatted
		}
	}
	return files, nil
}

func projectRootContent(projectName, projectTemplate, projectModule string) string {
	if projectTemplate == "http" {
		return fmt.Sprintf(`package cmd

import (
	"fmt"
	"log"
 "context"
 "time"
 "os/signal"
 "syscall"
 "net/http"
 "errors"
 %q

	"github.com/gin-gonic/gin"
 "gorm.io/gorm"
 %q
)

func NewRouter() *gin.Engine {
	router := gin.Default()
 router.Use(telemetry.Middleware())
	router.GET("/healthz", func(c *gin.Context) {
		c.String(200, "ok\n")
	})
	return router
}

var moduleInitializers []func(*gorm.DB, *gin.Engine) error

func Run() error {
 ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
 defer stop()
 shutdown, err := telemetry.Setup(ctx)
 if err != nil { return err }
 defer func(){ flush, cancel := context.WithTimeout(context.Background(), 5*time.Second); defer cancel(); _ = shutdown(flush) }()
 db, err := database.OpenDatabase()
 if err != nil { return err }
 sqlDB, err := db.DB()
 if err != nil { return err }
 defer sqlDB.Close()
 router := NewRouter()
 for _, initialize := range moduleInitializers {
  if err := initialize(db, router); err != nil { return err }
 }
 addr := ":8080"
	fmt.Printf("%%s listening on %%s\n", %q, addr)
	server := &http.Server{Addr: addr, Handler: router, ReadHeaderTimeout: 10*time.Second}
 done := make(chan error, 1)
 go func(){ done <- server.ListenAndServe() }()
 select {
 case err := <-done: if errors.Is(err, http.ErrServerClosed) { return nil }; return err
 case <-ctx.Done():
  closing, cancel := context.WithTimeout(context.Background(), 10*time.Second); defer cancel()
  return server.Shutdown(closing)
 }
}

func Execute() {
	if err := Run(); err != nil {
		log.Fatal(err)
	}
}
`, projectModule+"/internal/platform/telemetry", projectModule+"/internal/platform/database", projectName)
	}

	return fmt.Sprintf(`package cmd

import ("fmt"; "log"; "context"; "time"; %q; %q)

func Run() error {
 shutdown, err := telemetry.Setup(context.Background())
 if err != nil { return err }
 defer func(){ ctx, cancel := context.WithTimeout(context.Background(),5*time.Second); defer cancel(); _ = shutdown(ctx) }()
 db, err := database.OpenDatabase()
 if err != nil { return err }
 sqlDB, err := db.DB()
 if err != nil { return err }
 defer sqlDB.Close()
 fmt.Println(%q)
 return nil
}
func Execute(){ if err := Run(); err != nil { log.Fatal(err) } }
`, projectModule+"/internal/platform/database", projectModule+"/internal/platform/telemetry", "Welcome to "+projectName)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize modkit in the current project",
	Long: `Initialize a Go project from a project template.

Available templates:
  http     HTTP server with router and /healthz endpoint (default)
  console  console application with a command entrypoint`,
	Example: `  modkit init
  modkit init --template http
  modkit init --template console`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := os.Stat(config.DefaultFilename); err == nil {
			return fmt.Errorf("%s already exists", config.DefaultFilename)
		}

		cfg := config.Default()
		projectName := strings.TrimSpace(initName)
		if projectName == "" && dryRun {
			projectName = cfg.Project.Name
		}
		if projectName == "" {
			var err error
			projectName, err = promptProjectName()
			if err != nil {
				return err
			}
		}
		projectSlug := naming.DirectoryName(projectName)
		if projectSlug == "" {
			return fmt.Errorf("project name cannot be empty")
		}
		cfg.Project.Name = projectName
		cfg.Project.Module = "github.com/your-org/" + projectSlug
		cfg.Project.Template = initTemplate

		files, err := projectFiles(cfg, initTemplate)
		if err != nil {
			return err
		}
		configData, err := yaml.Marshal(cfg)
		if err != nil {
			return err
		}
		files[config.DefaultFilename] = configData
		if err := fileplan.Write(files, nil, dryRun, cmd.OutOrStdout()); err != nil {
			return err
		}
		if dryRun {
			return nil
		}

		fmt.Printf("Created %s\n", config.DefaultFilename)
		fmt.Printf("Created %s, go.mod, and cmd/\n", "main.go")
		return nil
	},
}

var initTemplate string
var initName string

func promptProjectName() (string, error) {
	defaultName := config.Default().Project.Name
	fmt.Printf("Project name [%s]: ", defaultName)
	name, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && strings.TrimSpace(name) == "" {
		return "", fmt.Errorf("read project name: %w", err)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return defaultName, nil
	}
	return name, nil
}

func init() {
	initCmd.Flags().StringVar(&initTemplate, "template", "http", "project template: http or console")
	initCmd.Flags().StringVar(&initName, "name", "", "project name (skips the interactive prompt)")
}

const databaseContent = `package database

import (
 "os"
 "context"
 "fmt"
 "PROJECT_MODULE/internal/platform/telemetry"
 "gorm.io/driver/postgres"
 "gorm.io/gorm"
)

// OpenDatabase opens the shared database. Replace the dialector to use another engine.
func OpenDatabase() (*gorm.DB, error) { return OpenDatabaseContext(context.Background()) }

func OpenDatabaseContext(ctx context.Context) (*gorm.DB, error) {
 _, span := telemetry.Start(ctx,"platform.database.OpenDatabase")
 defer span.End()
 dsn := os.Getenv("DB_DSN")
 if dsn == "" { return nil, fmt.Errorf("DB_DSN is required (PostgreSQL connection string)") }
 db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
 if err != nil { return nil, err }
 sqlDB, err := db.DB()
 if err != nil { return nil, err }
 sqlDB.SetMaxOpenConns(10)
 return db, nil
}
`

func writeGoSource(path, content string) error {
	formatted, err := format.Source([]byte(content))
	if err != nil {
		return fmt.Errorf("format %s: %w", path, err)
	}
	return os.WriteFile(path, formatted, 0o644)
}
