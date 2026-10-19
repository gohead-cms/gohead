package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/gohead-cms/gohead/internal/accounts"
	"github.com/gohead-cms/gohead/pkg/migrations"
	"github.com/gohead-cms/gohead/pkg/seed"

	"github.com/spf13/cobra"
)

var adminCmd = &cobra.Command{
	Use:   "admin",
	Short: "Manage administrator accounts",
}

var adminCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an administrator",
	Long: `Creates an account with the admin role.

The password is read from standard input (--password-stdin) and needs at
least 12 characters.

  echo "$PASSWORD" | gohead admin create --username alice --email alice@example.com --password-stdin`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		username, _ := cmd.Flags().GetString("username")
		email, _ := cmd.Flags().GetString("email")
		if username == "" || email == "" {
			return fmt.Errorf("--username and --email are required")
		}
		password, err := readPassword(cmd)
		if err != nil {
			return err
		}
		_, db, err := openDatabase(configFlag(cmd))
		if err != nil {
			return err
		}
		if err := migrations.Check(db); err != nil {
			return err
		}
		seed.SeedRoles()
		if _, err := accounts.CreateAdmin(username, email, password); err != nil {
			return err
		}
		fmt.Printf("Administrator %q created.\n", username)
		return nil
	},
}

var adminPasswordCmd = &cobra.Command{
	Use:   "set-password",
	Short: "Replace a user's password (to recover an admin account)",
	Long: `Replaces the password of --username, read from standard input
(--password-stdin). Sessions already open stay valid until they expire.

  echo "$PASSWORD" | gohead admin set-password --username alice --password-stdin`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		username, _ := cmd.Flags().GetString("username")
		if username == "" {
			return fmt.Errorf("--username is required")
		}
		password, err := readPassword(cmd)
		if err != nil {
			return err
		}
		_, db, err := openDatabase(configFlag(cmd))
		if err != nil {
			return err
		}
		if err := migrations.Check(db); err != nil {
			return err
		}
		if err := accounts.SetPassword(username, password); err != nil {
			return err
		}
		fmt.Printf("Password of %q replaced.\n", username)
		return nil
	},
}

// readPassword reads the password from stdin. Never from a flag (shell
// history) nor implicitly from the environment: GOHEAD_ADMIN_PASSWORD is the
// bootstrap admin's password and is set in every pod that has it.
func readPassword(cmd *cobra.Command) (string, error) {
	if fromStdin, _ := cmd.Flags().GetBool("password-stdin"); !fromStdin {
		return "", fmt.Errorf("give the password on standard input with --password-stdin")
	}
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && line == "" {
		return "", fmt.Errorf("could not read the password from stdin: %w", err)
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func init() {
	adminCmd.PersistentFlags().StringP("config", "c", "config.yaml", "Path to the configuration file")
	adminCmd.PersistentFlags().String("username", "", "account username")
	adminCmd.PersistentFlags().Bool("password-stdin", false, "read the password from standard input")
	adminCreateCmd.Flags().String("email", "", "account email")
	adminCmd.AddCommand(adminCreateCmd, adminPasswordCmd)
	rootCmd.AddCommand(adminCmd)
}
