package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var dryRun bool

var rootCmd = &cobra.Command{
	Use:   "modkit",
	Short: "DDD module scaffolding CLI",
	Long:  "modkit generates consistent DDD modules from reusable templates.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "preview generated file changes without writing or connecting to services")
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(moduleCmd)
	rootCmd.AddCommand(systemCmd)
	rootCmd.AddCommand(versionCmd)
}
