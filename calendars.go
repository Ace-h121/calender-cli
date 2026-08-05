package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"google.golang.org/api/calendar/v3"
)

var calendarsCmd = &cobra.Command{
	Use:     "calendars",
	Aliases: []string{"cal"},
	Short:   "Manage calendars",
}

var calendarsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List the user's calendars",
	RunE: func(cmd *cobra.Command, args []string) error {
		srv, err := newService()
		if err != nil {
			return err
		}
		max, _ := cmd.Flags().GetInt("max")
		resp, err := srv.CalendarList.List().MaxResults(int64(max)).Do()
		if err != nil {
			return fmt.Errorf("unable to list calendars: %w", err)
		}
		if len(resp.Items) == 0 {
			fmt.Println("No calendars found.")
			return nil
		}
		for _, c := range resp.Items {
			role := c.AccessRole
			if role == "" {
				role = "owner"
			}
			fmt.Printf("%-30s %-10s %s\n", c.Id, role, orDefault(c.Summary, "(untitled)"))
		}
		return nil
	},
}

var calendarsShowCmd = &cobra.Command{
	Use:   "show <calendarId>",
	Short: "Show details of a calendar",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		srv, err := newService()
		if err != nil {
			return err
		}
		c, err := srv.Calendars.Get(args[0]).Do()
		if err != nil {
			return fmt.Errorf("unable to get calendar: %w", err)
		}
		printCalendar(c)
		return nil
	},
}

var calendarsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a calendar",
	RunE: func(cmd *cobra.Command, args []string) error {
		srv, err := newService()
		if err != nil {
			return err
		}
		summary, _ := cmd.Flags().GetString("summary")
		if summary == "" {
			return fmt.Errorf("--summary is required")
		}
		c := &calendar.Calendar{Summary: summary}
		if v, _ := cmd.Flags().GetString("description"); v != "" {
			c.Description = v
		}
		if v, _ := cmd.Flags().GetString("timezone"); v != "" {
			c.TimeZone = v
		}
		if v, _ := cmd.Flags().GetString("location"); v != "" {
			c.Location = v
		}
		created, err := srv.Calendars.Insert(c).Do()
		if err != nil {
			return fmt.Errorf("unable to create calendar: %w", err)
		}
		fmt.Printf("Created calendar %s\n", created.Id)
		return nil
	},
}

var calendarsUpdateCmd = &cobra.Command{
	Use:   "update <calendarId>",
	Short: "Update a calendar",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		srv, err := newService()
		if err != nil {
			return err
		}
		c, err := srv.Calendars.Get(args[0]).Do()
		if err != nil {
			return fmt.Errorf("unable to get calendar: %w", err)
		}
		changed := func(name string) bool { return cmd.Flags().Changed(name) }
		if changed("summary") {
			c.Summary, _ = cmd.Flags().GetString("summary")
		}
		if changed("description") {
			c.Description, _ = cmd.Flags().GetString("description")
		}
		if changed("timezone") {
			c.TimeZone, _ = cmd.Flags().GetString("timezone")
		}
		if changed("location") {
			c.Location, _ = cmd.Flags().GetString("location")
		}
		if _, err := srv.Calendars.Update(args[0], c).Do(); err != nil {
			return fmt.Errorf("unable to update calendar: %w", err)
		}
		fmt.Printf("Updated calendar %s\n", args[0])
		return nil
	},
}

var calendarsDeleteCmd = &cobra.Command{
	Use:   "delete <calendarId>",
	Short: "Delete a calendar",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		srv, err := newService()
		if err != nil {
			return err
		}
		if err := srv.Calendars.Delete(args[0]).Do(); err != nil {
			return fmt.Errorf("unable to delete calendar: %w", err)
		}
		fmt.Printf("Deleted calendar %s\n", args[0])
		return nil
	},
}

func init() {
	calendarsCmd.AddCommand(calendarsListCmd)
	calendarsCmd.AddCommand(calendarsShowCmd)
	calendarsCmd.AddCommand(calendarsCreateCmd)
	calendarsCmd.AddCommand(calendarsUpdateCmd)
	calendarsCmd.AddCommand(calendarsDeleteCmd)

	calendarsListCmd.Flags().IntP("max", "n", 100, "maximum number of calendars to return")

	calendarsCreateCmd.Flags().String("summary", "", "calendar name (required)")
	calendarsCreateCmd.Flags().String("description", "", "calendar description")
	calendarsCreateCmd.Flags().String("timezone", "", "timezone, e.g. America/Los_Angeles")
	calendarsCreateCmd.Flags().String("location", "", "geographic location")

	calendarsUpdateCmd.Flags().AddFlagSet(calendarsCreateCmd.Flags())
}

func printCalendar(c *calendar.Calendar) {
	fmt.Printf("ID:          %s\n", c.Id)
	fmt.Printf("Summary:     %s\n", c.Summary)
	fmt.Printf("Description: %s\n", c.Description)
	fmt.Printf("Timezone:    %s\n", c.TimeZone)
	fmt.Printf("Location:    %s\n", c.Location)
}
