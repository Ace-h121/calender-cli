package main

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// parseTime parses a flexible date/time string into a time.Time in the local
// location. Supported forms: RFC3339, "2006-01-02 15:04:05", "2006-01-02T15:04:05",
// "2006-01-02", "15:04", the words "now"/"today"/"tomorrow"/"yesterday", and
// relative durations such as "+2h", "-30m", "+1d", "+1w".
func parseTime(s string) (time.Time, error) {
	base := time.Now()
	return parseTimeBase(s, base)
}

func parseTimeBase(s string, base time.Time) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, errors.New("empty time string")
	}

	switch strings.ToLower(s) {
	case "now":
		return base, nil
	case "today":
		return midnight(base), nil
	case "tomorrow":
		return midnight(base.AddDate(0, 0, 1)), nil
	case "yesterday":
		return midnight(base.AddDate(0, 0, -1)), nil
	}

	if s[0] == '+' || s[0] == '-' {
		return parseRelative(s, base)
	}

	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04",
		"2006-01-02 15:04",
		"2006-01-02",
		"15:04",
	}
	for _, layout := range layouts {
		t, err := time.ParseInLocation(layout, s, time.Local)
		if err != nil {
			continue
		}
		if layout == "15:04" {
			t = time.Date(base.Year(), base.Month(), base.Day(),
				t.Hour(), t.Minute(), 0, 0, time.Local)
		}
		return t, nil
	}
	return time.Time{}, fmt.Errorf("could not parse time %q", s)
}

var relRe = regexp.MustCompile(`^([+-])(\d+)([smhdw])$`)

func parseRelative(s string, base time.Time) (time.Time, error) {
	m := relRe.FindStringSubmatch(s)
	if m == nil {
		return time.Time{}, fmt.Errorf("could not parse relative time %q", s)
	}
	amount, err := strconv.Atoi(m[2])
	if err != nil {
		return time.Time{}, fmt.Errorf("could not parse relative time %q: %w", s, err)
	}
	var d time.Duration
	switch m[3] {
	case "s":
		d = time.Duration(amount) * time.Second
	case "m":
		d = time.Duration(amount) * time.Minute
	case "h":
		d = time.Duration(amount) * time.Hour
	case "d":
		d = time.Duration(amount) * 24 * time.Hour
	case "w":
		d = time.Duration(amount) * 7 * 24 * time.Hour
	}
	if m[1] == "-" {
		d = -d
	}
	return base.Add(d), nil
}

func midnight(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
}

// endOfDay returns 23:59:59 on the given day's local midnight... actually the
// exclusive end of the day: the next day at 00:00.
func endOfDay(t time.Time) time.Time {
	return midnight(t).AddDate(0, 0, 1)
}
