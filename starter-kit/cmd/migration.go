package cmd

import (
	"fmt"
	"path/filepath"
	"regexp"

	"github.com/spf13/cobra"
	"github.com/CdeT-Comite-de-Transformacion/LOS_Docs/starter-kit/internal/migration"
	fileplan "github.com/CdeT-Comite-de-Transformacion/LOS_Docs/starter-kit/internal/scaffold"
)

func init() {
	parent := &cobra.Command{Use: "migration", Short: "Create, check and prepare SQL migration merges",
		Long: "SQL migrations use Goose. Apply them with go run ./tools/migrate up; inspect with status.\nMerge produces a review draft without changing applied history or resolving business SQL automatically."}
	valid := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)
	create := &cobra.Command{Use: "create <name>", Short: "Create a versioned Up/Down SQL draft", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !valid.MatchString(args[0]) {
				return fmt.Errorf("invalid migration name")
			}
			path, err := migration.NextPath(args[0])
			if err != nil {
				return err
			}
			if err := fileplan.Write(map[string][]byte{path: migration.Draft()}, nil, dryRun, cmd.OutOrStdout()); err != nil {
				return err
			}
			if !dryRun {
				fmt.Fprintln(cmd.OutOrStdout(), "Created", path, "(review SQL before applying)")
			}
			return nil
		}}
	check := &cobra.Command{Use: "check", Short: "Check names, duplicate versions, annotations and unresolved conflicts (offline)", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := migration.Check(); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Migration structure OK (SQL semantics and applied history require database validation)")
			return nil
		}}
	var sources []string
	merge := &cobra.Command{Use: "merge <name>", Short: "Prepare a combined review draft; preserve every source migration", Args: cobra.ExactArgs(1),
		Example: "  modkit migration merge reconcile-applications --source 20261001120000_branch-a.sql --source 20261001120000_branch-b.sql --dry-run",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !valid.MatchString(args[0]) {
				return fmt.Errorf("invalid merge name")
			}
			content, err := migration.Merge(sources)
			if err != nil {
				return err
			}
			path := filepath.Join("migrations", "merge", args[0]+".sql")
			if err := fileplan.Write(map[string][]byte{path: content}, nil, dryRun, cmd.OutOrStdout()); err != nil {
				return err
			}
			if !dryRun {
				fmt.Fprintln(cmd.OutOrStdout(), "Created review draft", path, "; source files and database history are unchanged")
			}
			return nil
		}}
	merge.Flags().StringArrayVar(&sources, "source", nil, "source SQL filename (repeat; order determines Up order)")
	parent.AddCommand(create, check, merge)
	rootCmd.AddCommand(parent)
}
