package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/your-org/modkit/internal/config"
	"github.com/your-org/modkit/internal/generator"
)

var moduleValidateCmd = &cobra.Command{
	Use:   "validate <module>",
	Short: "Validate the generated module structure",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(config.DefaultFilename)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		gen, err := generator.New(cfg)
		if err != nil {
			return err
		}
		if err := gen.ValidateModule(args[0]); err != nil {
			return err
		}

		fmt.Printf("Module %q is valid\n", args[0])
		return nil
	},
}
