package flag

import (
	"fmt"
	"time"
)

// TimeRange represents a temporal validity range
type TimeRange struct {
	From time.Time
	To   *time.Time
}

// RangesOverlap checks if two time ranges overlap
// Returns true if the ranges overlap, false otherwise
// A range is [From, To) where To can be nil (meaning infinity)
func RangesOverlap(a, b TimeRange) bool {
	// Check if a.From is before b ends (or b has no end)
	aStartsBeforeBEnds := b.To == nil || a.From.Before(*b.To)

	// Check if b.From is before a ends (or a has no end)
	bStartsBeforeAEnds := a.To == nil || b.From.Before(*a.To)

	return aStartsBeforeBEnds && bStartsBeforeAEnds
}

// IsValidInRange checks if a given time is within the validity range
// Returns true if 'at' is >= validFrom and < validTo (or validTo is nil)
func IsValidInRange(at, validFrom time.Time, validTo *time.Time) bool {
	// at must be >= validFrom
	if at.Before(validFrom) {
		return false
	}

	// If validTo is set, at must be before it
	if validTo != nil && !at.Before(*validTo) {
		return false
	}

	return true
}

// ValidateTemporalRange validates a temporal range
func ValidateTemporalRange(validFrom time.Time, validTo *time.Time) error {
	if validFrom.IsZero() {
		return fmt.Errorf("validFrom is required")
	}

	if validTo != nil {
		if validTo.Before(validFrom) || validTo.Equal(validFrom) {
			return fmt.Errorf("validTo must be after validFrom")
		}
	}

	return nil
}
