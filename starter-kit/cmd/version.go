package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// version is injected at build time using Go linker flags.
var version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print modkit version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("modkit %s\n", version)
	},
}
