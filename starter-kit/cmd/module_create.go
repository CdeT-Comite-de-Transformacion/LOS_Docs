package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/CdeT-Comite-de-Transformacion/los-bff/starter-kit/internal/config"
	"github.com/CdeT-Comite-de-Transformacion/los-bff/starter-kit/internal/generator"
	"github.com/CdeT-Comite-de-Transformacion/los-bff/starter-kit/internal/module"
)

var moduleCreateOpts module.Options

var moduleCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new DDD module",
	Args:  cobra.ExactArgs(1),
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

		if err := gen.CreateModule(moduleCreateOpts); err != nil {
			return err
		}

		fmt.Printf("Created module %q in %s\n", moduleCreateOpts.Name, cfg.Modules.Root)
		return nil
	},
}

func init() {
	moduleCreateCmd.Flags().StringVar(&moduleCreateOpts.Type, "type", "", "module template type")
	moduleCreateCmd.Flags().BoolVar(&moduleCreateOpts.Force, "force", false, "overwrite existing generated files")
	moduleCreateCmd.Flags().BoolVar(&moduleCreateOpts.CreateAll, "create-all", false, "create all optional module elements")
}
