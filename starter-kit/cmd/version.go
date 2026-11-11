package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// version defaults to the current development baseline and is overridden for releases.
var version = "0.3.0"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print modkit version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("modkit %s\n", version)
	},
}
