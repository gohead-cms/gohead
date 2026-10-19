package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/gohead-cms/gohead/pkg/migrations"

	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Apply pending database migrations",
	Long: `Applies the pending database migrations and exits.

Migrations are versioned and recorded in the schema_migrations table; a
database lock lets several runners start at once. The API also applies them
at startup unless auto_migrate is false (GOHEAD_AUTO_MIGRATE=false), which is
how the Helm chart runs them once, in an init container.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		_, db, err := openDatabase(configFlag(cmd))
		if err != nil {
			return err
		}
		applied, err := migrations.Up(db)
		if err != nil {
			return err
		}
		if len(applied) == 0 {
			fmt.Println("Database schema is up to date.")
			return nil
		}
		for _, id := range applied {
			fmt.Println("applied", id)
		}
		return nil
	},
}

var migrateStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "List migrations and whether they ran",
	RunE: func(cmd *cobra.Command, _ []string) error {
		_, db, err := openDatabase(configFlag(cmd))
		if err != nil {
			return err
		}
		states, err := migrations.Status(db)
		if err != nil {
			return err
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tAPPLIED\tREVERSIBLE\tDESCRIPTION")
		pending := 0
		for _, s := range states {
			applied := "pending"
			if s.AppliedAt != nil {
				applied = s.AppliedAt.UTC().Format("2006-01-02 15:04:05Z")
			} else {
				pending++
			}
			fmt.Fprintf(w, "%s\t%s\t%v\t%s\n", s.ID, applied, s.Reversible, s.Description)
		}
		_ = w.Flush()
		if pending > 0 {
			return fmt.Errorf("%d pending migration(s)", pending)
		}
		return nil
	},
}

var migrateDownCmd = &cobra.Command{
	Use:   "down",
	Short: "Revert the last applied migrations",
	Long: `Reverts the last --steps applied migrations, newest first. Only
migrations with a Down step can be reverted; for the others, restore a
backup. Run it with the same GoHead version that applied them.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		steps, _ := cmd.Flags().GetInt("steps")
		if steps < 1 {
			return fmt.Errorf("--steps must be at least 1")
		}
		_, db, err := openDatabase(configFlag(cmd))
		if err != nil {
			return err
		}
		reverted, err := migrations.Down(db, steps)
		for _, id := range reverted {
			fmt.Println("reverted", id)
		}
		return err
	},
}

func configFlag(cmd *cobra.Command) string {
	path, _ := cmd.Flags().GetString("config")
	return path
}

func init() {
	migrateCmd.PersistentFlags().StringP("config", "c", "config.yaml", "Path to the configuration file")
	migrateDownCmd.Flags().Int("steps", 1, "number of migrations to revert")
	migrateCmd.AddCommand(migrateStatusCmd, migrateDownCmd)
	rootCmd.AddCommand(migrateCmd)
}
