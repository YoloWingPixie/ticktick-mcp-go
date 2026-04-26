package server

import (
	"testing"
	"time"
)

func TestParseTickTickDate(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"RFC3339 format", "2024-01-15T10:30:00Z", false},
		{"TickTick format with numeric tz", "2024-01-15T10:30:00+0000", false},
		{"invalid", "not-a-date", true},
		{"empty", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseTickTickDate(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTickTickDate(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestIsToday(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, now.Location())
	yesterday := today.AddDate(0, 0, -1)
	tomorrow := today.AddDate(0, 0, 1)

	tests := []struct {
		name string
		t    time.Time
		want bool
	}{
		{"today", today, true},
		{"yesterday", yesterday, false},
		{"tomorrow", tomorrow, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isToday(tt.t, now); got != tt.want {
				t.Errorf("isToday(%v) = %v, want %v", tt.t, got, tt.want)
			}
		})
	}
}

func TestIsOverdue(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, now.Location())
	yesterday := today.AddDate(0, 0, -1)
	tomorrow := today.AddDate(0, 0, 1)

	tests := []struct {
		name string
		t    time.Time
		want bool
	}{
		{"yesterday", yesterday, true},
		{"today", today, false},
		{"tomorrow", tomorrow, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isOverdue(tt.t, now); got != tt.want {
				t.Errorf("isOverdue(%v) = %v, want %v", tt.t, got, tt.want)
			}
		})
	}
}

func TestIsWithinDays(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, now.Location())

	tests := []struct {
		name string
		t    time.Time
		days int
		want bool
	}{
		{"today within 7 days", today, 7, true},
		{"6 days from now within 7 days", today.AddDate(0, 0, 6), 7, true},
		{"8 days from now within 7 days", today.AddDate(0, 0, 8), 7, false},
		{"yesterday within 7 days", today.AddDate(0, 0, -1), 7, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isWithinDays(tt.t, now, tt.days); got != tt.want {
				t.Errorf("isWithinDays(%v, %d) = %v, want %v", tt.t, tt.days, got, tt.want)
			}
		})
	}
}
