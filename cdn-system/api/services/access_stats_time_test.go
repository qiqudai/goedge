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

