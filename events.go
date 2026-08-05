package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/api/calendar/v3"
)

var eventsCmd = &cobra.Command{
	Use:     "events",
	Aliases: []string{"e"},
	Short:   "Manage calendar events",
}

type listParams struct {
	calendar string
	query    string
	orderBy  string
	max      int
	from     string
	to       string
	date     string
}

func listEvents(srv *calendar.Service, p listParams) error {
	if p.date != "" {
		d, err := parseTime(p.date)
		if err != nil {
			return fmt.Errorf("invalid --date: %w", err)
		}
		p.from = d.Format(time.RFC3339)
		p.to = endOfDay(d).Format(time.RFC3339)
	}

	call := srv.Events.List(p.calendar).
		ShowDeleted(false).
		SingleEvents(true).
		MaxResults(int64(p.max)).
		OrderBy(p.orderBy)
	if p.query != "" {
		call = call.Q(p.query)
	}
	if p.from != "" {
		t, err := parseTime(p.from)
		if err != nil {
			return fmt.Errorf("invalid --from: %w", err)
		}
		call = call.TimeMin(t.Format(time.RFC3339))
	}
	if p.to != "" {
		t, err := parseTime(p.to)
		if err != nil {
			return fmt.Errorf("invalid --to: %w", err)
		}
		call = call.TimeMax(t.Format(time.RFC3339))
	}

	events, err := call.Do()
	if err != nil {
		return fmt.Errorf("unable to list events: %w", err)
	}
	printEvents(events.Items)
	return nil
}

var eventsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List events",
	Long:  "List events for the given calendar, optionally filtered by time range, date, or text query.",
	RunE: func(cmd *cobra.Command, args []string) error {
		srv, err := newService()
		if err != nil {
			return err
		}
		p := listParams{}
		p.calendar, _ = cmd.Flags().GetString("calendar")
		p.query, _ = cmd.Flags().GetString("query")
		p.orderBy, _ = cmd.Flags().GetString("order-by")
		p.max, _ = cmd.Flags().GetInt("max")
		p.from, _ = cmd.Flags().GetString("from")
		p.to, _ = cmd.Flags().GetString("to")
		p.date, _ = cmd.Flags().GetString("date")
		return listEvents(srv, p)
	},
}

var eventsTodayCmd = &cobra.Command{
	Use:   "today",
	Short: "List today's events",
	RunE: func(cmd *cobra.Command, args []string) error {
		srv, err := newService()
		if err != nil {
			return err
		}
		p := listParams{calendar: "primary", orderBy: "startTime", max: 10, date: "today"}
		if c, _ := cmd.Flags().GetString("calendar"); c != "" {
			p.calendar = c
		}
		return listEvents(srv, p)
	},
}

var eventsShowCmd = &cobra.Command{
	Use:   "show <eventId>",
	Short: "Show details of a single event",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		srv, err := newService()
		if err != nil {
			return err
		}
		cal, _ := cmd.Flags().GetString("calendar")
		e, err := srv.Events.Get(cal, args[0]).Do()
		if err != nil {
			return fmt.Errorf("unable to get event: %w", err)
		}
		printEventDetails(e)
		return nil
	},
}

var eventsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an event",
	RunE: func(cmd *cobra.Command, args []string) error {
		srv, err := newService()
		if err != nil {
			return err
		}
		cal, _ := cmd.Flags().GetString("calendar")
		event, err := eventFromFlags(cmd, true)
		if err != nil {
			return err
		}
		call := srv.Events.Insert(cal, event)
		if notify, _ := cmd.Flags().GetBool("notify"); notify {
			call = call.SendUpdates("all")
		}
		created, err := call.Do()
		if err != nil {
			return fmt.Errorf("unable to create event: %w", err)
		}
		fmt.Printf("Created event %s\n", created.Id)
		fmt.Printf("  Link: %s\n", created.HtmlLink)
		return nil
	},
}

var eventsUpdateCmd = &cobra.Command{
	Use:   "update <eventId>",
	Short: "Update an event",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		srv, err := newService()
		if err != nil {
			return err
		}
		cal, _ := cmd.Flags().GetString("calendar")
		existing, err := srv.Events.Get(cal, args[0]).Do()
		if err != nil {
			return fmt.Errorf("unable to get event: %w", err)
		}
		patched, err := eventFromFlags(cmd, false)
		if err != nil {
			return err
		}

		applyPatch(existing, patched, cmd)

		call := srv.Events.Update(cal, args[0], existing)
		if notify, _ := cmd.Flags().GetBool("notify"); notify {
			call = call.SendUpdates("all")
		}
		updated, err := call.Do()
		if err != nil {
			return fmt.Errorf("unable to update event: %w", err)
		}
		fmt.Printf("Updated event %s\n", updated.Id)
		return nil
	},
}

var eventsDeleteCmd = &cobra.Command{
	Use:   "delete <eventId>",
	Short: "Delete an event",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		srv, err := newService()
		if err != nil {
			return err
		}
		cal, _ := cmd.Flags().GetString("calendar")
		call := srv.Events.Delete(cal, args[0])
		if notify, _ := cmd.Flags().GetBool("notify"); notify {
			call = call.SendUpdates("all")
		}
		if err := call.Do(); err != nil {
			return fmt.Errorf("unable to delete event: %w", err)
		}
		fmt.Printf("Deleted event %s\n", args[0])
		return nil
	},
}

var eventsQuickAddCmd = &cobra.Command{
	Use:   "quickadd \"<text>\"",
	Short: "Create an event from natural language",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		srv, err := newService()
		if err != nil {
			return err
		}
		cal, _ := cmd.Flags().GetString("calendar")
		call := srv.Events.QuickAdd(cal, args[0])
		if notify, _ := cmd.Flags().GetBool("notify"); notify {
			call = call.SendUpdates("all")
		}
		e, err := call.Do()
		if err != nil {
			return fmt.Errorf("unable to quick add event: %w", err)
		}
		fmt.Printf("Created event %s: %s\n", e.Id, e.Summary)
		return nil
	},
}

func init() {
	eventsCmd.AddCommand(eventsListCmd)
	eventsCmd.AddCommand(eventsTodayCmd)
	eventsCmd.AddCommand(eventsShowCmd)
	eventsCmd.AddCommand(eventsCreateCmd)
	eventsCmd.AddCommand(eventsUpdateCmd)
	eventsCmd.AddCommand(eventsDeleteCmd)
	eventsCmd.AddCommand(eventsQuickAddCmd)

	eventsCmd.PersistentFlags().StringP("calendar", "c", "primary", "calendar ID to operate on")
	eventsCmd.PersistentFlags().Bool("notify", false, "send update notifications to attendees")

	eventsListCmd.Flags().String("from", "", "start of range (RFC3339 or flexible time, e.g. \"2026-08-05\" or \"tomorrow\")")
	eventsListCmd.Flags().String("to", "", "end of range")
	eventsListCmd.Flags().String("date", "", "list events on a single day (overrides --from/--to)")
	eventsListCmd.Flags().StringP("query", "q", "", "free-text search in title/location/description")
	eventsListCmd.Flags().IntP("max", "n", 10, "maximum number of events to return")
	eventsListCmd.Flags().String("order-by", "startTime", "sort field: startTime or updated")

	eventsCreateCmd.Flags().String("title", "", "event title (required)")
	eventsCreateCmd.Flags().String("start", "", "start time (required unless --all-day)")
	eventsCreateCmd.Flags().String("end", "", "end time (defaults to start + 1h)")
	eventsCreateCmd.Flags().Bool("all-day", false, "create an all-day event")
	eventsCreateCmd.Flags().String("location", "", "event location")
	eventsCreateCmd.Flags().String("description", "", "event description")
	eventsCreateCmd.Flags().String("attendees", "", "comma-separated attendee emails")
	eventsCreateCmd.Flags().String("timezone", "", "timezone for the event (defaults to local)")
	eventsCreateCmd.Flags().String("reminders", "", "reminders like \"popup=10,email=30\"")
	eventsCreateCmd.Flags().StringArray("rrule", nil, "recurrence rule(s) like \"RRULE:FREQ=WEEKLY;COUNT=10\"")

	eventsUpdateCmd.Flags().AddFlagSet(eventsCreateCmd.Flags())
	eventsShowCmd.Flags().String("calendar", "primary", "calendar ID")
	eventsDeleteCmd.Flags().Bool("notify", false, "send update notifications to attendees")
}

// eventFromFlags builds an event from the create/update flag set. When
// requireRequired is true, missing required fields are errors.
func eventFromFlags(cmd *cobra.Command, requireRequired bool) (*calendar.Event, error) {
	title, _ := cmd.Flags().GetString("title")
	startStr, _ := cmd.Flags().GetString("start")
	endStr, _ := cmd.Flags().GetString("end")
	allDay, _ := cmd.Flags().GetBool("all-day")
	location, _ := cmd.Flags().GetString("location")
	description, _ := cmd.Flags().GetString("description")
	attendees, _ := cmd.Flags().GetString("attendees")
	timezone, _ := cmd.Flags().GetString("timezone")
	remindersStr, _ := cmd.Flags().GetString("reminders")
	rrules, _ := cmd.Flags().GetStringArray("rrule")

	if requireRequired {
		if title == "" {
			return nil, fmt.Errorf("--title is required")
		}
		if startStr == "" && !allDay {
			return nil, fmt.Errorf("--start is required (or use --all-day)")
		}
	}
	if allDay && startStr == "" {
		return nil, fmt.Errorf("--start is required for all-day events")
	}

	event := &calendar.Event{Summary: title}
	if location != "" {
		event.Location = location
	}
	if description != "" {
		event.Description = description
	}
	if len(rrules) > 0 {
		event.Recurrence = rrules
	}
	if attendees != "" {
		var list []*calendar.EventAttendee
		for _, email := range strings.Split(attendees, ",") {
			email = strings.TrimSpace(email)
			if email != "" {
				list = append(list, &calendar.EventAttendee{Email: email})
			}
		}
		event.Attendees = list
	}
	if remindersStr != "" {
		reminders, err := parseReminders(remindersStr)
		if err != nil {
			return nil, err
		}
		event.Reminders = &calendar.EventReminders{
			UseDefault: false,
			Overrides:  reminders,
		}
	}

	if startStr != "" {
		start, err := parseTime(startStr)
		if err != nil {
			return nil, fmt.Errorf("invalid --start: %w", err)
		}
		if allDay {
			event.Start = &calendar.EventDateTime{Date: start.Format("2006-01-02")}
			if endStr != "" {
				end, err := parseTime(endStr)
				if err != nil {
					return nil, fmt.Errorf("invalid --end: %w", err)
				}
				event.End = &calendar.EventDateTime{Date: end.Format("2006-01-02")}
			} else {
				event.End = &calendar.EventDateTime{Date: endOfDay(start).Format("2006-01-02")}
			}
		} else {
			tz := timezone
			if tz == "" {
				tz = time.Local.String()
			}
			end := start.Add(time.Hour)
			if endStr != "" {
				var err error
				end, err = parseTime(endStr)
				if err != nil {
					return nil, fmt.Errorf("invalid --end: %w", err)
				}
			}
			event.Start = &calendar.EventDateTime{
				DateTime: start.Format(time.RFC3339),
				TimeZone: tz,
			}
			event.End = &calendar.EventDateTime{
				DateTime: end.Format(time.RFC3339),
				TimeZone: tz,
			}
		}
	}
	return event, nil
}

// applyPatch copies fields that were explicitly set on the command line from
// the patched event onto the existing event, so unspecified fields survive.
func applyPatch(existing, patch *calendar.Event, cmd *cobra.Command) {
	changed := func(name string) bool { return cmd.Flags().Changed(name) }
	if changed("title") {
		existing.Summary = patch.Summary
	}
	if changed("location") {
		existing.Location = patch.Location
	}
	if changed("description") {
		existing.Description = patch.Description
	}
	if changed("attendees") {
		existing.Attendees = patch.Attendees
	}
	if changed("reminders") {
		existing.Reminders = patch.Reminders
	}
	if changed("rrule") {
		existing.Recurrence = patch.Recurrence
	}
	if changed("start") || changed("end") || changed("all-day") {
		existing.Start = patch.Start
		existing.End = patch.End
	}
}

func parseReminders(s string) ([]*calendar.EventReminder, error) {
	var out []*calendar.EventReminder
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid reminder %q, expected method=minutes", part)
		}
		method := strings.ToLower(strings.TrimSpace(kv[0]))
		if method != "email" && method != "popup" && method != "sms" {
			return nil, fmt.Errorf("invalid reminder method %q (use email, popup, or sms)", method)
		}
		minutes, err := strconv.Atoi(strings.TrimSpace(kv[1]))
		if err != nil {
			return nil, fmt.Errorf("invalid reminder minutes in %q", part)
		}
		out = append(out, &calendar.EventReminder{
			Method:  method,
			Minutes: int64(minutes),
		})
	}
	return out, nil
}

func eventRange(e *calendar.Event) (start, end time.Time, allDay bool) {
	if e.Start.DateTime != "" {
		s, err := time.Parse(time.RFC3339, e.Start.DateTime)
		if err == nil {
			start = s
		}
		if e.End.DateTime != "" {
			if et, err := time.Parse(time.RFC3339, e.End.DateTime); err == nil {
				end = et
			}
		}
		return start, end, false
	}
	if e.Start.Date != "" {
		if s, err := time.Parse("2006-01-02", e.Start.Date); err == nil {
			start = s
		}
		if e.End.Date != "" {
			if et, err := time.Parse("2006-01-02", e.End.Date); err == nil {
				end = et
			}
		}
		return start, end, true
	}
	return time.Now(), time.Now(), false
}

func printEvents(items []*calendar.Event) {
	if len(items) == 0 {
		fmt.Println("No events found.")
		return
	}
	for _, e := range items {
		start, end, allDay := eventRange(e)
		var line string
		if allDay {
			line = fmt.Sprintf("%s  All day  %s", start.Format("2006-01-02"), orDefault(e.Summary, "(untitled)"))
		} else {
			line = fmt.Sprintf("%s  %s-%s  %s",
				start.Format("2006-01-02 15:04"),
				start.Format("15:04"),
				end.Format("15:04"),
				orDefault(e.Summary, "(untitled)"))
		}
		if e.Status == "cancelled" {
			line += "  [cancelled]"
		}
		fmt.Println(line)
	}
}

func printEventDetails(e *calendar.Event) {
	start, end, allDay := eventRange(e)
	fmt.Printf("ID:          %s\n", e.Id)
	fmt.Printf("Status:      %s\n", e.Status)
	fmt.Printf("Title:       %s\n", orDefault(e.Summary, "(untitled)"))
	fmt.Printf("Location:    %s\n", e.Location)
	if allDay {
		fmt.Printf("When:        All day, %s", start.Format("2006-01-02"))
		if !end.IsZero() {
			fmt.Printf(" to %s", end.Format("2006-01-02"))
		}
		fmt.Println()
	} else {
		fmt.Printf("Start:       %s\n", start.Format("Mon Jan 02 2006 15:04"))
		if !end.IsZero() {
			fmt.Printf("End:         %s\n", end.Format("Mon Jan 02 2006 15:04"))
		}
	}
	if len(e.Recurrence) > 0 {
		fmt.Printf("Recurrence:  %s\n", strings.Join(e.Recurrence, " | "))
	}
	if e.Reminders != nil {
		parts := make([]string, 0, len(e.Reminders.Overrides))
		for _, r := range e.Reminders.Overrides {
			parts = append(parts, fmt.Sprintf("%s %dm", r.Method, r.Minutes))
		}
		if e.Reminders.UseDefault {
			parts = append(parts, "use default")
		}
		fmt.Printf("Reminders:   %s\n", strings.Join(parts, ", "))
	}
	if len(e.Attendees) > 0 {
		var names []string
		for _, a := range e.Attendees {
			status := a.ResponseStatus
			if status == "" {
				status = "pending"
			}
			names = append(names, fmt.Sprintf("%s (%s)", a.Email, status))
		}
		fmt.Printf("Attendees:   %s\n", strings.Join(names, ", "))
	}
	if e.Description != "" {
		fmt.Printf("Description:\n%s\n", e.Description)
	}
	if e.HtmlLink != "" {
		fmt.Printf("Link:        %s\n", e.HtmlLink)
	}
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
