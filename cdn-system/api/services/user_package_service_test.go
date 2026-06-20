package services

import (
	"cdn-api/models"
	"testing"
	"time"
)

func TestIsUserPackageExpired(t *testing.T) {
	now := time.Date(2026, 5, 28, 8, 0, 0, 0, time.UTC)

	cases := []struct {
		name  string
		endAt time.Time
		want  bool
	}{
		{name: "zero", want: false},
		{name: "future", endAt: now.Add(time.Minute), want: false},
		{name: "exact", endAt: now, want: true},
		{name: "past", endAt: now.Add(-time.Minute), want: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := isUserPackageExpired(models.UserPackage{EndAt: tc.endAt}, now)
			if got != tc.want {
				t.Fatalf("isUserPackageExpired = %v, want %v", got, tc.want)
			}
		})
	}
}
