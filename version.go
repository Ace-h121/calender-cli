package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var version = "0.1.0"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version)
	},
}

// runUpcoming is the default action when the binary is invoked without a
// subcommand: it lists upcoming events, mirroring the original program.
func runUpcoming() error {
	srv, err := newService()
	if err != nil {
		return err
	}
	p := listParams{calendar: "primary", orderBy: "startTime", max: 10, from: "now"}
	return listEvents(srv, p)
}
