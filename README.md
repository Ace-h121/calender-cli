# calender-cli

A feature-complete command-line client for the Google Calendar API, written in Go.

> **Intended use case: AI agents.** This CLI is primarily designed to be driven
> by AI agents — it wraps the Calendar API behind a stable, scriptable command
> surface so agents can read and manage a user's calendar without hand-coding
> API calls or OAuth. See [skill.md](skill.md) for an agent-oriented usage
> guide.

## Features

- **Events**: list, show, create, update, delete, and natural-language quick-add
- **Calendars**: list, show, create, update, delete
- **Auth**: login, logout, and status (including granted scopes)
- Flexible date/time parsing (`"2026-08-05"`, `"14:30"`, `"tomorrow"`, `"+2h"`, …)
- Loopback OAuth flow — the authorization code is captured automatically

## Prerequisites

1. A Google Cloud project with the **Google Calendar API** enabled.
2. An OAuth client of type **Desktop app**.
3. Your `client-secret.json` placed at:

   ```
   ~/.config/calender-cli/client-secret.json
   ```

## Install

```sh
go build -o calender-cli .
# optionally: go install .
```

## Authenticate

```sh
calender-cli auth login
```

This prints a URL. Open it in a browser, approve the consent screen, and the
CLI captures the authorization code automatically and saves it to
`~/.config/calender-cli/token.json`.

```sh
calender-cli auth status    # show token location, expiry, granted scopes
calender-cli auth logout    # revoke and remove the stored token
```

If you ever see `insufficient scope` errors, run `calender-cli auth login`
again — the scope changed to full calendar access so write operations work.

## Usage

Running `calender-cli` with no arguments lists your next 10 upcoming events.

### Events

```sh
# List events
calender-cli events list                     # next 10 events
calender-cli events list --from tomorrow     # events starting tomorrow
calender-cli events list --to "2026-12-31"
calender-cli events list --date 2026-08-05   # all events on one day
calender-cli events list --query "dentist"   # free-text search
calender-cli events list --max 50
calender-cli events list -c <calendar-id>    # non-default calendar

# Today's events
calender-cli events today

# Show one event
calender-cli events show <eventId>

# Create
calender-cli events create --title "Dentist" --start "tomorrow 14:00" --end "+3h"
calender-cli events create --title "Vacation" --start 2026-08-10 --end 2026-08-17 --all-day
calender-cli events create --title "Standup" --start "09:30" --rrule "RRULE:FREQ=WEEKLY;COUNT=10"
calender-cli events create --title "Review" --start "+1h" --end "+2h" \
    --location "Room 3" --description "Q3 review" \
    --attendees "a@example.com,b@example.com" --reminders "popup=10,email=30" \
    --timezone America/Los_Angeles --notify

# Update (only flags you pass are changed)
calender-cli events update <eventId> --title "New title" --start "2026-08-06 15:00"

# Delete
calender-cli events delete <eventId>
calender-cli events delete <eventId> --notify   # notify attendees

# Quick add (natural language)
calender-cli events quickadd "Lunch with Sam Friday at noon"
```

### Calendars

```sh
calender-cli calendars list
calender-cli calendars show <calendarId>
calender-cli calendars create --summary "Work" --timezone America/Los_Angeles
calender-cli calendars update <calendarId> --summary "Work 2026"
calender-cli calendars delete <calendarId>
```

### Time formats

`--from`, `--to`, `--date`, `--start`, and `--end` accept:

| Input | Meaning |
| --- | --- |
| `2026-08-05` | that date, local midnight |
| `2026-08-05T14:30:00` / RFC3339 | exact instant |
| `14:30` | today at 14:30 |
| `today`, `tomorrow`, `yesterday`, `now` | relative day / instant |
| `+2h`, `-30m`, `+1d`, `+1w` | relative to now |

## Project layout

| File | Purpose |
| --- | --- |
| `main.go` | root command and service plumbing |
| `auth.go` | OAuth flow and `auth` subcommands |
| `events.go` | event commands and formatting |
| `calendars.go` | calendar commands |
| `timeparse.go` | flexible time parsing |
| `version.go` | version + default "upcoming" action |
| `skill.md` | agent-oriented usage guide (intended for AI agents) |

## License

MIT
