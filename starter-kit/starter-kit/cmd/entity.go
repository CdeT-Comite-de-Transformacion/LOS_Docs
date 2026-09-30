package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/your-org/modkit/internal/config"
	"github.com/your-org/modkit/internal/generator"
)

func init() {
	var moduleName string
	var fields []string
	var force bool
	entityCmd := &cobra.Command{
		Use:   "entity",
		Short: "Manage entities within modules",
		Long: `Manage domain entities inside existing modules.

Use create to generate an entity, repository, application service, and GORM CRUD.
HTTP projects also receive CRUD routes registered in the application router.
Run "modkit entity create --help" for fields and generation options.`,
		Example: `  modkit entity create customer --module sales
  modkit entity create customer --module sales --field name:string --field active:bool`,
	}
	createCmd := &cobra.Command{
		Use: "create <name>", Short: "Create an entity with GORM persistence and CRUD",
		Long: `Create an entity inside a module previously created with "modkit module create".

Generates domain, application, and infrastructure layers under
<modules.root>/<module>/entities/<name>/. HTTP projects also receive POST and GET
routes at /<module>/<name>, and GET, PUT, DELETE routes at /<module>/<name>/:id.

List routes accept limit (default 20, maximum 100) and offset (default 0),
and return items, limit, offset, and total.

Each entity includes a client-supplied string ID. Add fields with repeated
--field name:type flags. Supported types: string, bool, int, int64, uint,
uint64, float64. Use hyphen-separated names such as first-name for compound fields.
Existing files are protected unless --force is supplied.`,
		Example: `  modkit module create sales
  modkit entity create customer --module sales
  modkit entity create product --module sales --field name:string --field price:float64
  modkit entity create customer --module sales --field active:bool --force`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(config.DefaultFilename)
			if err != nil {
				return err
			}
			gen, err := generator.New(cfg)
			if err != nil {
				return err
			}
			if err := gen.CreateEntity(moduleName, args[0], fields, force); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Created entity %q in module %q\n", args[0], moduleName)
			return nil
		},
	}
	createCmd.Flags().StringVar(&moduleName, "module", "", "existing module name (required)")
	createCmd.Flags().StringArrayVar(&fields, "field", nil, "entity field name:type (repeatable)")
	createCmd.Flags().BoolVar(&force, "force", false, "overwrite generated entity files")
	_ = createCmd.MarkFlagRequired("module")
	entityCmd.AddCommand(createCmd)
	rootCmd.AddCommand(entityCmd)
}
