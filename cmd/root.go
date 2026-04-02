package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/MaximusBenjamin/terminal-pomodoro/internal/app"
	"github.com/MaximusBenjamin/terminal-pomodoro/internal/store"
	"github.com/spf13/cobra"
)

var appVersion string

// SetVersion sets the application version (injected at build time).
func SetVersion(v string) {
	appVersion = v
	rootCmd.Version = appVersion
}

var rootCmd = &cobra.Command{
	Use:   "tpom",
	Short: "A minimalist pomodoro timer for the terminal",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := store.New()
		if err != nil {
			return fmt.Errorf("initializing store: %w", err)
		}
		defer s.Close()

		m := app.New(s)
		p := tea.NewProgram(m, tea.WithAltScreen())
		_, err = p.Run()
		return err
	},
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
