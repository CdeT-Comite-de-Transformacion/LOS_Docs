package cmd

import (
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"
	"github.com/your-org/modkit/internal/config"
)

var moduleListCmd = &cobra.Command{
	Use:   "list",
	Short: "List generated modules",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(config.DefaultFilename)
		if err != nil {
			return err
		}

		entries, err := os.ReadDir(cfg.Modules.Root)
		if os.IsNotExist(err) {
			fmt.Println("No modules yet")
			return nil
		}
		if err != nil {
			return err
		}

		var names []string
		for _, entry := range entries {
			if entry.IsDir() {
				names = append(names, entry.Name())
			}
		}
		sort.Strings(names)

		if len(names) == 0 {
			fmt.Println("No modules yet")
			return nil
		}

		for _, name := range names {
			fmt.Println(name)
		}
		return nil
	},
}
