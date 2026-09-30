package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var systemCmd = &cobra.Command{
	Use:   "system",
	Short: "Print the detected operating system and architecture",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("os: %s\narchitecture: %s\n", runtime.GOOS, runtime.GOARCH)
	},
}
