package services

import (
	"testing"
	"time"
)

func TestParseCKTimeString_ParsesNaiveAsUTC(t *testing.T) {
	origLocal := time.Local
	time.Local = time.FixedZone("CST", 8*3600)
	defer func() {
		time.Local = origLocal
	}()

	parsed, err := parseCKTimeString("2026-03-30 15:05:58")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	wantUnix := time.Date(2026, 3, 30, 15, 5, 58, 0, time.UTC).Unix()
	if parsed.Unix() != wantUnix {
		t.Fatalf("unexpected unix, got=%d want=%d", parsed.Unix(), wantUnix)
	}
}

func TestBuildBucketSeries_MatchesUTCBucketWithLocalRange(t *testing.T) {
	origLocal := time.Local
	time.Local = time.FixedZone("CST", 8*3600)
	defer func() {
		time.Local = origLocal
	}()

	rng := StatsRange{
		Start:       time.Date(2026, 3, 30, 22, 28, 0, 0, time.Local),
		End:         time.Date(2026, 3, 30, 22, 30, 0, 0, time.Local),
		Bucket:      time.Minute,
		LabelFormat: "15:04",
	}

	bucket, err := parseCKTimeString("2026-03-30 14:29:00")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	series := BuildBucketSeries(rng, []AccessBucket{
		{Bucket: bucket, Requests: 7},
	})

	if len(series.Requests) != 3 {
		t.Fatalf("unexpected series length: %d", len(series.Requests))
	}
	if series.Requests[1] != 7 {
		t.Fatalf("bucket value mismatch at 22:29, got=%d want=7", series.Requests[1])
	}
}

func TestNormalizeSkew_KeepsShortRealtimeLag(t *testing.T) {
	nowTS := int64(1000)
	maxTS := nowTS - 120

	got := normalizeSkew(maxTS, nowTS)
	if got != -120*time.Second {
		t.Fatalf("unexpected skew, got=%s want=-2m0s", got)
	}
}

func TestAdjustAccessLogQueryRangeForSkew_ShiftsRealtimeWindow(t *testing.T) {
	now := time.Date(2026, 4, 29, 14, 30, 0, 0, time.UTC)
	start := now.Add(-10 * time.Minute)
	end := now

	adjustedStart, adjustedEnd, displayShift := adjustAccessLogQueryRangeForSkew(start, end, now, -35*time.Minute)

	if !adjustedStart.Equal(start.Add(-35 * time.Minute)) {
		t.Fatalf("unexpected adjusted start, got=%s", adjustedStart)
	}
	if !adjustedEnd.Equal(end.Add(-35 * time.Minute)) {
		t.Fatalf("unexpected adjusted end, got=%s", adjustedEnd)
	}
	if displayShift != 35*time.Minute {
		t.Fatalf("unexpected display shift, got=%s want=35m0s", displayShift)
	}
}

func TestAdjustAccessLogQueryRangeForSkew_DoesNotShiftHistoricalWindow(t *testing.T) {
	now := time.Date(2026, 4, 29, 14, 30, 0, 0, time.UTC)
	start := now.Add(-25 * time.Hour)
	end := start.Add(30 * time.Minute)

	adjustedStart, adjustedEnd, displayShift := adjustAccessLogQueryRangeForSkew(start, end, now, -35*time.Minute)

	if !adjustedStart.Equal(start) || !adjustedEnd.Equal(end) || displayShift != 0 {
		t.Fatalf("historical range should not shift, start=%s end=%s display=%s", adjustedStart, adjustedEnd, displayShift)
	}
}

func TestBuildBucketSeries_AlignsShiftedRealtimeBucket(t *testing.T) {
	now := time.Date(2026, 4, 29, 14, 30, 0, 0, time.UTC)
	rng := StatsRange{
		Start:       now.Add(-2 * time.Minute),
		End:         now,
		Bucket:      time.Minute,
		LabelFormat: "15:04",
	}
	_, _, displayShift := adjustAccessLogQueryRangeForSkew(rng.Start, rng.End, now, -35*time.Minute)
	queriedBucket := now.Add(-36 * time.Minute).Truncate(time.Minute)

	series := BuildBucketSeries(rng, []AccessBucket{
		{Bucket: queriedBucket.Add(displayShift), Requests: 9},
	})

	if len(series.Requests) != 3 {
		t.Fatalf("unexpected series length: %d", len(series.Requests))
	}
	if series.Requests[1] != 9 {
		t.Fatalf("shifted bucket value mismatch, got=%d want=9", series.Requests[1])
	}
}
