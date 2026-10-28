package cmd

import "github.com/spf13/cobra"

var moduleCmd = &cobra.Command{
	Use:   "module",
	Short: "Manage DDD modules",
}

func init() {
	moduleCmd.AddCommand(moduleCreateCmd)
	moduleCmd.AddCommand(moduleAddElementCmd)
	moduleCmd.AddCommand(moduleListCmd)
	moduleCmd.AddCommand(moduleValidateCmd)
}
