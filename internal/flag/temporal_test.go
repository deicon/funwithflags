package flag

import (
	"testing"
	"time"
)

func TestValidateTemporalRange(t *testing.T) {
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	later := now.Add(24 * time.Hour)
	earlier := now.Add(-24 * time.Hour)

	tests := []struct {
		name      string
		validFrom time.Time
		validTo   *time.Time
		wantErr   bool
	}{
		{
			name:      "valid open-ended range",
			validFrom: now,
			validTo:   nil,
			wantErr:   false,
		},
		{
			name:      "valid bounded range",
			validFrom: now,
			validTo:   &later,
			wantErr:   false,
		},
		{
			name:      "zero validFrom",
			validFrom: time.Time{},
			validTo:   nil,
			wantErr:   true,
		},
		{
			name:      "validTo before validFrom",
			validFrom: now,
			validTo:   &earlier,
			wantErr:   true,
		},
		{
			name:      "validTo equals validFrom",
			validFrom: now,
			validTo:   &now,
			wantErr:   true,
		},
		{
			name:      "validFrom in the past is allowed",
			validFrom: earlier,
			validTo:   &later,
			wantErr:   false,
		},
		{
			name:      "very short valid range (1 second)",
			validFrom: now,
			validTo:   timePtr(now.Add(time.Second)),
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTemporalRange(tt.validFrom, tt.validTo)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestRangesOverlap(t *testing.T) {
	t1 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	t3 := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	t4 := time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		a       TimeRange
		b       TimeRange
		overlap bool
	}{
		{
			name:    "no overlap: A before B",
			a:       TimeRange{From: t1, To: &t2},
			b:       TimeRange{From: t3, To: &t4},
			overlap: false,
		},
		{
			name:    "no overlap: B before A",
			a:       TimeRange{From: t3, To: &t4},
			b:       TimeRange{From: t1, To: &t2},
			overlap: false,
		},
		{
			name:    "adjacent ranges (A ends where B starts, half-open)",
			a:       TimeRange{From: t1, To: &t2},
			b:       TimeRange{From: t2, To: &t3},
			overlap: false,
		},
		{
			name:    "overlapping ranges",
			a:       TimeRange{From: t1, To: &t3},
			b:       TimeRange{From: t2, To: &t4},
			overlap: true,
		},
		{
			name:    "A contains B",
			a:       TimeRange{From: t1, To: &t4},
			b:       TimeRange{From: t2, To: &t3},
			overlap: true,
		},
		{
			name:    "B contains A",
			a:       TimeRange{From: t2, To: &t3},
			b:       TimeRange{From: t1, To: &t4},
			overlap: true,
		},
		{
			name:    "identical ranges",
			a:       TimeRange{From: t1, To: &t2},
			b:       TimeRange{From: t1, To: &t2},
			overlap: true,
		},
		{
			name:    "open-ended A overlaps with B",
			a:       TimeRange{From: t1, To: nil},
			b:       TimeRange{From: t2, To: &t3},
			overlap: true,
		},
		{
			name:    "open-ended B overlaps with A",
			a:       TimeRange{From: t2, To: &t3},
			b:       TimeRange{From: t1, To: nil},
			overlap: true,
		},
		{
			name:    "both open-ended",
			a:       TimeRange{From: t1, To: nil},
			b:       TimeRange{From: t2, To: nil},
			overlap: true,
		},
		{
			name:    "open-ended A, B entirely before A starts",
			a:       TimeRange{From: t3, To: nil},
			b:       TimeRange{From: t1, To: &t2},
			overlap: false,
		},
		{
			name:    "open-ended B, A entirely before B starts",
			a:       TimeRange{From: t1, To: &t2},
			b:       TimeRange{From: t3, To: nil},
			overlap: false,
		},
		{
			name:    "open-ended A, B ends exactly when A starts (half-open)",
			a:       TimeRange{From: t2, To: nil},
			b:       TimeRange{From: t1, To: &t2},
			overlap: false,
		},
		{
			name:    "same start time, different ends",
			a:       TimeRange{From: t1, To: &t2},
			b:       TimeRange{From: t1, To: &t3},
			overlap: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RangesOverlap(tt.a, tt.b)
			if got != tt.overlap {
				t.Errorf("RangesOverlap(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.overlap)
			}

			// Overlap is symmetric
			gotReversed := RangesOverlap(tt.b, tt.a)
			if gotReversed != tt.overlap {
				t.Errorf("RangesOverlap(%v, %v) reversed = %v, want %v", tt.b, tt.a, gotReversed, tt.overlap)
			}
		})
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}
