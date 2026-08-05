package main

import (
	"context"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

func configDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "calender-cli")
}

func tokenPath() string  { return filepath.Join(configDir(), "token.json") }
func secretPath() string { return filepath.Join(configDir(), "client-secret.json") }

func newService() (*calendar.Service, error) {
	config, err := loadConfig()
	if err != nil {
		return nil, err
	}
	client := getClient(config, tokenPath())
	return calendar.NewService(context.Background(), option.WithHTTPClient(client))
}

var rootCmd = &cobra.Command{
	Use:   "calender-cli",
	Short: "A command-line client for Google Calendar",
	Long:  "A feature-complete CLI for reading and managing Google Calendar events and calendars.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUpcoming()
	},
}

func init() {
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(eventsCmd)
	rootCmd.AddCommand(calendarsCmd)
	rootCmd.AddCommand(versionCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
