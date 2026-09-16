package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

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
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(moduleCmd)
	rootCmd.AddCommand(systemCmd)
	rootCmd.AddCommand(versionCmd)
}
