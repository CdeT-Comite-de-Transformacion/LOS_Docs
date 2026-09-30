package cmd

import (
	"bufio"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/your-org/modkit/internal/config"
	"github.com/your-org/modkit/internal/naming"
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

	for _, path := range []string{"main.go", filepath.Join("cmd", "root.go"), filepath.Join("cmd", "database.go"), "go.mod"} {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("%s already exists", path)
		}
	}

	if err := os.MkdirAll("cmd", 0o755); err != nil {
		return err
	}

	mainContent := fmt.Sprintf("package main\n\nimport \"%s/cmd\"\n\nfunc main() {\n\tcmd.Execute()\n}\n", projectModule)
	if err := writeGoSource("main.go", mainContent); err != nil {
		return err
	}

	rootContent := projectRootContent(cfg.Project.Name, projectTemplate)
	if err := writeGoSource(filepath.Join("cmd", "root.go"), rootContent); err != nil {
		return err
	}

	goModContent := fmt.Sprintf("module %s\n\ngo 1.24\n\nrequire (\n gorm.io/gorm v1.30.0\n gorm.io/driver/postgres v1.6.0\n)\n", projectModule)
	if projectTemplate == "http" {
		goModContent += "\nrequire github.com/gin-gonic/gin v1.11.0\n"
	}
	if err := os.WriteFile("go.mod", []byte(goModContent), 0o644); err != nil {
		return err
	}

	return writeGoSource(filepath.Join("cmd", "database.go"), databaseContent)
}

func projectRootContent(projectName, projectTemplate string) string {
	if projectTemplate == "http" {
		return fmt.Sprintf(`package cmd

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
 "gorm.io/gorm"
)

func NewRouter() *gin.Engine {
	router := gin.Default()
	router.GET("/healthz", func(c *gin.Context) {
		c.String(200, "ok\n")
	})
	return router
}

var moduleInitializers []func(*gorm.DB, *gin.Engine) error

func Run() error {
 db, err := OpenDatabase()
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
	return router.Run(addr)
}

func Execute() {
	if err := Run(); err != nil {
		log.Fatal(err)
	}
}
`, projectName)
	}

	return fmt.Sprintf(`package cmd

import ("fmt"; "log")

func Execute() {
 db, err := OpenDatabase()
 if err != nil { log.Fatal(err) }
 sqlDB, err := db.DB()
 if err != nil { log.Fatal(err) }
 defer sqlDB.Close()
 fmt.Println(%q)
}
`, "Welcome to "+projectName)
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
		if err := initProjectWithTemplate(cfg, initTemplate); err != nil {
			return err
		}
		if err := config.Save(config.DefaultFilename, cfg); err != nil {
			return err
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

const databaseContent = `package cmd

import (
 "os"
 "fmt"
 "gorm.io/driver/postgres"
 "gorm.io/gorm"
)

// OpenDatabase opens the shared database. Replace the dialector to use another engine.
func OpenDatabase() (*gorm.DB, error) {
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
