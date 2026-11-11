package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/CdeT-Comite-de-Transformacion/LOS_Docs/starter-kit/internal/config"
	"github.com/CdeT-Comite-de-Transformacion/LOS_Docs/starter-kit/internal/generator"
)

func init() {
	var moduleName string
	parent := &cobra.Command{Use: "platform", Short: "Manage optional infrastructure adapters",
		Long: "PostgreSQL is the default database. Add audit, redis, sns, sqs, s3/minio as needed.\nWithout --module adapters are installed at project level; with it, explicit module wiring is generated.\nNo external resources are provisioned or connected during generation."}
	add := &cobra.Command{Use: "add <audit|redis|sns|sqs|s3|minio>", Short: "Install an optional adapter and its Go dependencies", Args: cobra.ExactArgs(1),
		Example: "  modkit platform add redis --module origination --dry-run\n  modkit platform add audit --module origination\n  modkit platform add minio --module documents",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(config.DefaultFilename)
			if err != nil {
				return err
			}
			gen, err := generator.New(cfg)
			if err != nil {
				return err
			}
			gen.DryRun, gen.Output = dryRun, cmd.OutOrStdout()
			if err := gen.AddPlatform(args[0], moduleName); err != nil {
				return err
			}
			if !dryRun {
				fmt.Fprintln(cmd.OutOrStdout(), "Added", args[0], "; run go mod tidy and wire clients explicitly")
			}
			return nil
		}}
	add.Flags().StringVar(&moduleName, "module", "", "also generate integration for an existing module")

	var listModule string
	list := &cobra.Command{Use: "list", Short: "List available adapters and their installation status", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(config.DefaultFilename)
			if err != nil {
				return err
			}
			gen, err := generator.New(cfg)
			if err != nil {
				return err
			}
			status, err := gen.PlatformStatus(listModule)
			if err != nil {
				return err
			}
			for _, entry := range status {
				fmt.Fprintf(cmd.OutOrStdout(), "%s installed=%t", entry.Name, entry.Installed)
				if listModule != "" {
					fmt.Fprintf(cmd.OutOrStdout(), " module_enabled=%t", entry.Enabled)
				}
				fmt.Fprintln(cmd.OutOrStdout())
			}
			return nil
		}}
	list.Flags().StringVar(&listModule, "module", "", "show explicit module integration")
	parent.AddCommand(add, list)
	rootCmd.AddCommand(parent)
}
