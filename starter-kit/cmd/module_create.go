package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/CdeT-Comite-de-Transformacion/LOS_Docs/starter-kit/internal/config"
	"github.com/CdeT-Comite-de-Transformacion/LOS_Docs/starter-kit/internal/generator"
	"github.com/CdeT-Comite-de-Transformacion/LOS_Docs/starter-kit/internal/module"
)

var moduleCreateOpts module.Options

var moduleCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new DDD module",
	Long: `Create a business module with shared layers and no implicit entity.
Entities added later generate CRUD by default. --no-crud disables CRUD for
all entities in this module. --create-all adds optional integrations but
does not override --no-crud. Existing modules are never recreated.
Use entity create or module add-element to extend them.`,
	Example: `  modkit module create origination
  modkit module create underwriting --no-crud
  modkit entity create application --module origination
  modkit entity create applicant --module origination --no-crud`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(config.DefaultFilename)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		moduleCreateOpts.Name = args[0]
		if moduleCreateOpts.Type == "" {
			moduleCreateOpts.Type = cfg.Defaults.Template
		}

		gen, err := generator.New(cfg)
		if err != nil {
			return err
		}

		gen.DryRun, gen.Output = dryRun, cmd.OutOrStdout()
		if err := gen.CreateModule(moduleCreateOpts); err != nil {
			return err
		}

		if dryRun {
			return nil
		}
		fmt.Printf("Created module %q in %s\n", moduleCreateOpts.Name, cfg.Modules.Root)
		return nil
	},
}

func init() {
	moduleCreateCmd.Flags().BoolVar(&moduleCreateOpts.NoCRUD, "no-crud", false, "disable CRUD generation for entities in this module")
	moduleCreateCmd.Flags().StringVar(&moduleCreateOpts.Type, "type", "", "module template type")
	moduleCreateCmd.Flags().BoolVar(&moduleCreateOpts.CreateAll, "create-all", false, "create all optional module elements")
}
