package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

const version = "0.3.0"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print modkit version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("modkit %s\n", version)
	},
}
