package services

import (
	"strings"
	"time"
)

const statsTimeLayout = "2006-01-02 15:04:05"

// StatsRange defines a time window and bucket size for stats queries.
type StatsRange struct {
	Start       time.Time
	End         time.Time
	Bucket      time.Duration
	LabelFormat string
}

func ResolveStatsRange(rangeKey, startRaw, endRaw string, now time.Time) StatsRange {
	key := strings.ToLower(strings.TrimSpace(rangeKey))
	if key == "" {
		key = "30min"
	}
	switch key {
	case "today":
		start := beginningOfDay(now)
		return StatsRange{Start: start, End: now, Bucket: time.Hour, LabelFormat: "15:00"}
	case "yesterday":
		start := beginningOfDay(now).AddDate(0, 0, -1)
		end := endOfDay(start)
		return StatsRange{Start: start, End: end, Bucket: time.Hour, LabelFormat: "15:00"}
	case "7d", "7days", "7-day", "7":
		start := beginningOfDay(now).AddDate(0, 0, -6)
		return StatsRange{Start: start, End: now, Bucket: 24 * time.Hour, LabelFormat: "01-02"}
	case "30d", "30days", "30-day", "30":
		start := beginningOfDay(now).AddDate(0, 0, -29)
		return StatsRange{Start: start, End: now, Bucket: 24 * time.Hour, LabelFormat: "01-02"}
	case "last_month":
		start := beginningOfMonth(now).AddDate(0, -1, 0)
		end := endOfMonth(start)
		return StatsRange{Start: start, End: end, Bucket: 24 * time.Hour, LabelFormat: "01-02"}
	case "10min":
		return StatsRange{Start: now.Add(-10 * time.Minute), End: now, Bucket: time.Minute, LabelFormat: "15:04"}
	case "1h":
		return StatsRange{Start: now.Add(-1 * time.Hour), End: now, Bucket: time.Minute, LabelFormat: "15:04"}
	case "custom":
		if start, end, ok := parseCustomRange(startRaw, endRaw, now.Location()); ok {
			return buildCustomRange(start, end)
		}
	}
	return StatsRange{Start: now.Add(-30 * time.Minute), End: now, Bucket: time.Minute, LabelFormat: "15:04"}
}

func parseCustomRange(startRaw, endRaw string, loc *time.Location) (time.Time, time.Time, bool) {
	startRaw = strings.TrimSpace(startRaw)
	endRaw = strings.TrimSpace(endRaw)
	if startRaw == "" || endRaw == "" {
		return time.Time{}, time.Time{}, false
	}
	start, err1 := time.ParseInLocation(statsTimeLayout, startRaw, loc)
	end, err2 := time.ParseInLocation(statsTimeLayout, endRaw, loc)
	if err1 != nil || err2 != nil || end.Before(start) {
		return time.Time{}, time.Time{}, false
	}
	return start, end, true
}

func buildCustomRange(start, end time.Time) StatsRange {
	duration := end.Sub(start)
	if duration <= time.Hour {
		return StatsRange{Start: start, End: end, Bucket: time.Minute, LabelFormat: "15:04"}
	}
	if duration <= 24*time.Hour {
		return StatsRange{Start: start, End: end, Bucket: time.Hour, LabelFormat: "15:00"}
	}
	return StatsRange{Start: start, End: end, Bucket: 24 * time.Hour, LabelFormat: "01-02"}
}

func AlignToBucket(ts time.Time, bucket time.Duration) time.Time {
	if bucket >= 24*time.Hour {
		return beginningOfDay(ts)
	}
	return ts.Truncate(bucket)
}

func beginningOfDay(ts time.Time) time.Time {
	return time.Date(ts.Year(), ts.Month(), ts.Day(), 0, 0, 0, 0, ts.Location())
}

func beginningOfMonth(ts time.Time) time.Time {
	return time.Date(ts.Year(), ts.Month(), 1, 0, 0, 0, 0, ts.Location())
}

func endOfDay(ts time.Time) time.Time {
	return time.Date(ts.Year(), ts.Month(), ts.Day(), 23, 59, 59, 0, ts.Location())
}

func endOfMonth(ts time.Time) time.Time {
	startOfMonth := beginningOfMonth(ts)
	startNext := startOfMonth.AddDate(0, 1, 0)
	return startNext.Add(-time.Second)
}
