package cmd

import (
	"fmt"
	"os"
	"syscall"

	"github.com/MaximusBenjamin/terminal-pomodoro/internal/api"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to your tpom account",
	RunE: func(cmd *cobra.Command, args []string) error {
		email, _ := cmd.Flags().GetString("email")
		if email == "" {
			fmt.Fprint(os.Stderr, "Email: ")
			fmt.Scanln(&email)
		}
		if email == "" {
			return fmt.Errorf("email is required")
		}

		fmt.Fprint(os.Stderr, "Password: ")
		pw, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return fmt.Errorf("reading password: %w", err)
		}
		password := string(pw)
		if password == "" {
			return fmt.Errorf("password is required")
		}

		fmt.Println("Logging in...")
		token, err := api.Login(email, password)
		if err != nil {
			return fmt.Errorf("login failed: %w", err)
		}

		if err := api.SaveAuth(token); err != nil {
			return fmt.Errorf("saving auth: %w", err)
		}

		fmt.Printf("Logged in as %s\n", token.Email)
		return nil
	},
}

func init() {
	loginCmd.Flags().StringP("email", "e", "", "Email address")
	rootCmd.AddCommand(loginCmd)
}
