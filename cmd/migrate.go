package cmd

import (
	"fmt"

	"github.com/MaximusBenjamin/terminal-pomodoro/internal/api"
	"github.com/MaximusBenjamin/terminal-pomodoro/internal/store"
	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Upload local SQLite data to Supabase",
	Long:  "One-time migration: reads habits and sessions from ~/.pomo/pomo.db and uploads them to your Supabase account.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !api.IsLoggedIn() {
			return fmt.Errorf("not logged in — run: tpom login")
		}

		// Open local SQLite
		s, err := store.New()
		if err != nil {
			return fmt.Errorf("opening local database: %w", err)
		}
		defer s.Close()

		// Open Supabase client
		c, err := api.NewClient()
		if err != nil {
			return fmt.Errorf("creating API client: %w", err)
		}
		defer c.Close()

		// --- Migrate habits ---
		localHabits, err := s.ListHabits()
		if err != nil {
			return fmt.Errorf("reading local habits: %w", err)
		}

		remoteHabits, err := c.ListHabits()
		if err != nil {
			return fmt.Errorf("reading remote habits: %w", err)
		}

		// Map remote habits by name to avoid duplicates
		remoteByName := make(map[string]int)
		for _, h := range remoteHabits {
			remoteByName[h.Name] = h.ID
		}

		// Map local habit ID -> remote habit ID
		habitIDMap := make(map[int]int)

		for _, h := range localHabits {
			if remoteID, exists := remoteByName[h.Name]; exists {
				habitIDMap[h.ID] = remoteID
				fmt.Printf("  habit %q already exists (remote id %d)\n", h.Name, remoteID)
			} else {
				newID, err := c.AddHabit(h.Name, h.Color)
				if err != nil {
					return fmt.Errorf("creating habit %q: %w", h.Name, err)
				}
				habitIDMap[h.ID] = newID
				fmt.Printf("  habit %q created (remote id %d)\n", h.Name, newID)
			}
		}

		// --- Migrate sessions ---
		localSessions, err := s.RecentSessions(10000) // get all
		if err != nil {
			return fmt.Errorf("reading local sessions: %w", err)
		}

		if len(localSessions) == 0 {
			fmt.Println("No sessions to migrate.")
			return nil
		}

		fmt.Printf("\nMigrating %d sessions...\n", len(localSessions))

		migrated := 0
		skipped := 0
		for _, sess := range localSessions {
			remoteHabitID, ok := habitIDMap[sess.HabitID]
			if !ok {
				skipped++
				continue
			}

			err := c.CreateManualSession(
				remoteHabitID,
				sess.StartTime,
				sess.EndTime,
				sess.ActualSeconds,
			)
			if err != nil {
				fmt.Printf("  warning: failed to migrate session %d: %v\n", sess.ID, err)
				skipped++
				continue
			}
			migrated++
		}

		fmt.Printf("\nDone! %d sessions migrated, %d skipped.\n", migrated, skipped)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)
}
