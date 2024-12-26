package stringutils_test

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vbyazilim/basichttpdebugger/internal/stringutils"
)

func TestFormatDate(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		date     *time.Time
		expected string
	}{
		{
			name:     "Day with leading zeros",
			format:   "%d",
			date:     parseDate("2024-12-26T15:04:05"),
			expected: "26",
		},
		{
			name:     "Day without leading zeros",
			format:   "%j",
			date:     parseDate("2024-12-26T15:04:05"),
			expected: "26",
		},
		{
			name:     "Abbreviated weekday",
			format:   "%D",
			date:     parseDate("2024-12-26T15:04:05"),
			expected: "Thu",
		},
		{
			name:     "Full weekday",
			format:   "%l",
			date:     parseDate("2024-12-26T15:04:05"),
			expected: "Thursday",
		},
		{
			name:     "Week number",
			format:   "%W",
			date:     parseDate("2024-12-26T15:04:05"),
			expected: "52",
		},
		{
			name:     "Month with leading zeros",
			format:   "%m",
			date:     parseDate("2024-12-26T15:04:05"),
			expected: "12",
		},
		{
			name:     "Month without leading zeros",
			format:   "%n",
			date:     parseDate("2024-12-26T15:04:05"),
			expected: "12",
		},
		{
			name:     "Year, 2 digits",
			format:   "%y",
			date:     parseDate("2024-12-26T15:04:05"),
			expected: "24",
		},
		{
			name:     "Year, 4 digits",
			format:   "%Y",
			date:     parseDate("2024-12-26T15:04:05"),
			expected: "2024",
		},
		{
			name:     "12-hour format, no leading zeros",
			format:   "%g",
			date:     parseDate("2024-12-26T15:04:05"),
			expected: "3",
		},
		{
			name:     "12-hour format 00:00, no leading zeros",
			format:   "%g",
			date:     parseDate("2024-12-26T00:00:00"),
			expected: "12",
		},
		{
			name:     "24-hour format",
			format:   "%H",
			date:     parseDate("2024-12-26T15:04:05"),
			expected: "15",
		},
		{
			name:     "Minutes",
			format:   "%i",
			date:     parseDate("2024-12-26T15:04:05"),
			expected: "04",
		},
		{
			name:     "Seconds",
			format:   "%s",
			date:     parseDate("2024-12-26T15:04:05"),
			expected: "05",
		},
		{
			name:     "Microseconds",
			format:   "%u",
			date:     parseDateWithNanoseconds("2024-12-26T15:04:05.123456"),
			expected: "123456",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := stringutils.FormatDate(test.format, test.date)
			assert.Equal(t, test.expected, result)
		})
	}
}

// Helper function to parse a date string
func parseDate(dateString string) *time.Time {
	t, err := time.Parse("2006-01-02T15:04:05", dateString)
	if err != nil {
		panic(err)
	}
	return &t
}

// Helper function to parse a date string with nanoseconds
func parseDateWithNanoseconds(dateString string) *time.Time {
	t, err := time.Parse("2006-01-02T15:04:05.000000", dateString)
	if err != nil {
		panic(err)
	}
	return &t
}

func TestGetFormattedFilename(t *testing.T) {
	tests := []struct {
		name     string
		format   string
		req      *http.Request
		expected string
	}{
		{
			name:   "Basic formatting with sanitized hostname and URL",
			format: "%Y-%m-%d-{hostname}-{url}.raw",
			req: &http.Request{
				Host: "localhost:9000",
				URL:  &url.URL{Path: "/example/path"},
			},
			expected: "2024-12-26-localhost_9000-_example_path.raw",
		},
		{
			name:   "Special characters in hostname and URL",
			format: "%Y-%m-%d-%F-{hostname}-{url}.raw",
			req: &http.Request{
				Host: "host<>:\"/\\|?*name",
				URL:  &url.URL{Path: "/example/<invalid>?query"},
			},
			expected: "2024-12-26-December-host_name-_example_invalid_query.raw",
		},
		{
			name:   "Empty format string",
			format: "",
			req: &http.Request{
				Host: "localhost:9000",
				URL:  &url.URL{Path: "/example/path"},
			},
			expected: "",
		},
		{
			name:   "Only hostname placeholder",
			format: "{hostname}.raw",
			req: &http.Request{
				Host: "localhost:9000",
				URL:  &url.URL{Path: "/example/path"},
			},
			expected: "localhost_9000.raw",
		},
		{
			name:   "Only URL placeholder",
			format: "{url}.raw",
			req: &http.Request{
				Host: "localhost",
				URL:  &url.URL{Path: "/example/path"},
			},
			expected: "_example_path.raw",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := stringutils.GetFormattedFilename(test.format, test.req)
			assert.Equal(t, test.expected, result)
		})
	}
}
