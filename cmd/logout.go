package cmd

import (
	"fmt"

	"github.com/MaximusBenjamin/terminal-pomodoro/internal/api"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out of your tpom account",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := api.Logout(); err != nil {
			return err
		}
		fmt.Println("Logged out successfully.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}
