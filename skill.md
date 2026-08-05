---
name: calender-cli
description: Read, search, create, update, and delete Google Calendar events and calendars using the calender-cli command-line tool. Use this whenever the user needs help with their schedule, upcoming events, or calendar management tasks.
---

# calender-cli — Google Calendar for AI agents

`calender-cli` is a Go CLI that wraps the Google Calendar API. Its primary
intended use case is to let AI agents read and manage a user's calendar without
hand-coding API calls or OAuth. This skill tells an agent exactly how to use it
safely and correctly.

## Capabilities

| Task | Command |
| --- | --- |
| List upcoming / filtered events | `calender-cli events list` |
| Today's events | `calender-cli events today` |
| Event details | `calender-cli events show <id>` |
| Create an event | `calender-cli events create ...` |
| Update an event | `calender-cli events update <id> ...` |
| Delete an event | `calender-cli events delete <id>` |
| Natural-language quick add | `calender-cli events quickadd "<text>"` |
| List calendars | `calender-cli calendars list` |
| Calendar CRUD | `calender-cli calendars show/create/update/delete` |
| Auth management | `calender-cli auth login/logout/status` |

Run `calender-cli --help` or `<command> --help` for authoritative flag
reference — never guess flags you have not verified.

## Auth (do this first)

The CLI must be authenticated before any calendar operation:

1. Check `calender-cli auth status`. If it says "Not authenticated" or the
   granted scopes do not include `.../auth/calendar` (full scope), the agent
   must ask the user to run `calender-cli auth login` — it opens a browser
   consent flow that cannot be completed non-interactively.
2. Write operations require the full `calendar` scope. A read-only token will
   fail with `insufficient scope`; the fix is `calender-cli auth login`.

## Conventions and rules

- **Verify before you act.** For destructive or state-changing operations
  (update, delete), confirm the exact event/calendar ID with the user first.
- **Get IDs first.** `events list` output does not print event IDs. Use
  `calender-cli events show <id>` output, or ask the user, or list then show
  to confirm the right target before mutating.
- **Update is patch semantics.** `events update` only changes the flags you
  pass. To move an event, pass `--start`/`--end`; to rename, pass `--title`.
  Never assume unspecified fields are reset — they are preserved.
- **All-day events** are created with `--all-day` and `--start <date>`
  (`--end` is exclusive). Do not pass times for all-day events.
- **Timed events** use `--start`/`--end`; if `--end` is omitted it defaults to
  start + 1 hour.
- **Search** with `--query` matches the free-text `q` parameter of the API
  (title, description, location).
- **Recurrence** uses `--rrule` with full RRULE strings, e.g.
  `--rrule "RRULE:FREQ=WEEKLY;COUNT=10"`.
- **Attendees** are a comma-separated list of emails, `--attendees a@x.com,b@y.com`.
- **Reminders** use `method=minutes`, comma-separated, methods limited to
  `popup`, `email`, `sms`, e.g. `--reminders "popup=10,email=30"`.
- **`--notify`** sends update notifications to attendees; use it when a change
  is significant, omit it for silent housekeeping.
- **`-c <calendarId>`** targets a non-default calendar (default is `primary`).

## Time format cheat sheet

`--from`, `--to`, `--date`, `--start`, and `--end` all accept flexible input:

| Input | Meaning |
| --- | --- |
| `now`, `today`, `tomorrow`, `yesterday` | relative day / instant |
| `14:30` | today at 14:30 |
| `2026-08-05` | that date at local midnight |
| `2026-08-05T14:30:00` / RFC3339 | exact instant |
| `+2h`, `-30m`, `+1d`, `+1w` | relative to now |

Prefer explicit dates/RFC3339 for reproducible behavior; use relative forms
(`tomorrow`, `+1d`) only when the user's intent is clearly relative.

## Example agent flows

**"What's my schedule this week?"**
```
calender-cli events list --from today --to "+1w"
```

**"Do I have anything Friday?"**
```
calender-cli events list --date "tomorrow"   # or the concrete date
```

**"Find my dentist appointment"**
```
calender-cli events list --query dentist
calender-cli events show <id>   # to confirm before acting
```

**"Move my dentist appointment to Friday at 10"**
```
calender-cli events show <id>   # confirm the right event first
calender-cli events update <id> --start "2026-08-07 10:00" --end "10:45" --notify
```

**"Add a lunch with Sam tomorrow at noon"**
```
calender-cli events quickadd "Lunch with Sam tomorrow at noon"
```

## Troubleshooting

- **`insufficient scope`** — token lacks full `calendar` scope; run
  `calender-cli auth login`.
- **`Error: ...` with a usage block** — the command exited non-zero; read the
  error before retrying; flags may be misspelled or the wrong subcommand used.
- **`Unable to read client secret file`** — `~/.config/calender-cli/client-secret.json`
  is missing; ask the user to place it there.
- **Auth URLs open a browser** — never attempt to complete OAuth by hand; if
  `auth login` is needed, hand control back to the user.
