package cli

import (
	"fmt"
	"time"

	"github.com/itda-work/zap/internal/issue"
)

// DateFilter represents date filtering options
type DateFilter struct {
	Today bool
	Since string
	Until string
	Year  string
	Month string
	Date  string
	Days  int
	Weeks int
}

// IsEmpty returns true if no date filter is set
func (f *DateFilter) IsEmpty() bool {
	return !f.Today && f.Since == "" && f.Until == "" && f.Year == "" && f.Month == "" && f.Date == "" && f.Days == 0 && f.Weeks == 0
}

// GetDateRange returns the start and end time based on filter options
func (f *DateFilter) GetDateRange() (start, end time.Time, err error) {
	now := time.Now()
	loc := now.Location()

	// Default: no filter (zero time)
	start = time.Time{}
	end = time.Time{}

	if f.Today {
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
		end = start.Add(24 * time.Hour)
		return
	}

	if f.Days > 0 {
		end = now
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -f.Days+1)
		return
	}

	if f.Weeks > 0 {
		end = now
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -f.Weeks*7+1)
		return
	}

	if f.Date != "" {
		t, parseErr := time.ParseInLocation("2006-01-02", f.Date, loc)
		if parseErr != nil {
			return start, end, fmt.Errorf("invalid date format: %s (expected YYYY-MM-DD)", f.Date)
		}
		start = t
		end = t.Add(24 * time.Hour)
		return
	}

	if f.Month != "" {
		t, parseErr := time.ParseInLocation("2006-01", f.Month, loc)
		if parseErr != nil {
			return start, end, fmt.Errorf("invalid month format: %s (expected YYYY-MM)", f.Month)
		}
		start = t
		end = t.AddDate(0, 1, 0)
		return
	}

	if f.Year != "" {
		t, parseErr := time.ParseInLocation("2006", f.Year, loc)
		if parseErr != nil {
			return start, end, fmt.Errorf("invalid year format: %s (expected YYYY)", f.Year)
		}
		start = t
		end = t.AddDate(1, 0, 0)
		return
	}

	if f.Since != "" {
		t, parseErr := time.ParseInLocation("2006-01-02", f.Since, loc)
		if parseErr != nil {
			return start, end, fmt.Errorf("invalid since date format: %s (expected YYYY-MM-DD)", f.Since)
		}
		start = t
	}

	if f.Until != "" {
		t, parseErr := time.ParseInLocation("2006-01-02", f.Until, loc)
		if parseErr != nil {
			return start, end, fmt.Errorf("invalid until date format: %s (expected YYYY-MM-DD)", f.Until)
		}
		// Until is inclusive, so add 1 day
		end = t.Add(24 * time.Hour)
	}

	return
}

// FilterIssuesByDate filters issues based on date range
func FilterIssuesByDate(issues []*issue.Issue, filter *DateFilter) ([]*issue.Issue, error) {
	if filter.IsEmpty() {
		return issues, nil
	}

	start, end, err := filter.GetDateRange()
	if err != nil {
		return nil, err
	}

	var results []*issue.Issue
	for _, iss := range issues {
		// Check created_at or updated_at
		if matchesDateRange(iss.CreatedAt, start, end) || matchesDateRange(iss.UpdatedAt, start, end) {
			results = append(results, iss)
		}
	}

	return results, nil
}

// matchesDateRange checks if a time falls within the given range
func matchesDateRange(t time.Time, start, end time.Time) bool {
	if t.IsZero() {
		return false
	}

	// If start is set and time is before start, no match
	if !start.IsZero() && t.Before(start) {
		return false
	}

	// If end is set and time is after or equal to end, no match
	if !end.IsZero() && !t.Before(end) {
		return false
	}

	return true
}
