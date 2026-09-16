package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/CdeT-Comite-de-Transformacion/los-bff/starter-kit/internal/config"
	"github.com/CdeT-Comite-de-Transformacion/los-bff/starter-kit/internal/generator"
	"github.com/CdeT-Comite-de-Transformacion/los-bff/starter-kit/internal/module"
)

var moduleAddElementOpts module.Options

var moduleAddElementCmd = &cobra.Command{
	Use:   "add-element <module> <element>",
	Short: "Add an optional element to an existing module",
	Long: `Add one optional element to an existing module.

Available elements:
	http           HTTP handler boundary
	messaging      event publisher and consumer contract
	outbox         transactional event storage contract
	threading      coordinated goroutine workers
	observability  command and event instrumentation hooks`,
	Example: `  modkit module add-element credit-request http
	modkit module add-element credit-request messaging
	modkit module add-element credit-request outbox
	modkit module add-element credit-request threading
	modkit module add-element credit-request observability`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(config.DefaultFilename)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		if args[1] != "http" && args[1] != "messaging" && args[1] != "observability" && args[1] != "outbox" && args[1] != "threading" {
			return fmt.Errorf("unknown element %q; use http, messaging, outbox, threading, or observability", args[1])
		}

		moduleAddElementOpts.Name = args[0]
		moduleAddElementOpts.Elements = []string{args[1]}
		if moduleAddElementOpts.Type == "" {
			moduleAddElementOpts.Type = cfg.Defaults.Template
		}

		gen, err := generator.New(cfg)
		if err != nil {
			return err
		}
		if err := gen.AddModuleElements(moduleAddElementOpts); err != nil {
			return err
		}
		fmt.Printf("Added %s to module %q\n", args[1], args[0])
		return nil
	},
}

func init() {
	moduleAddElementCmd.Flags().StringVar(&moduleAddElementOpts.Type, "type", "", "module template type")
	moduleAddElementCmd.Flags().BoolVar(&moduleAddElementOpts.Force, "force", false, "overwrite existing generated files")
}
