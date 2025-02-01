package stringutils

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	twelveHour = 12
	thousand   = 1000
	modHundred = 100
)

func illegalChars() *regexp.Regexp {
	return regexp.MustCompile(`[<>:"/\\|?*\s]+`)
}

// NowFunc is used to get the current time. It defaults to time.Now, but
// can be overridden in tests.
var NowFunc = time.Now

// UserHomeDirFunc is used to get the user's home directory.
// It defaults to os.UserHomeDir but can be overridden in tests.
var UserHomeDirFunc = os.UserHomeDir

// FormatDate formats given date as Django's format date style (most of them).
// https://docs.djangoproject.com/en/5.1/ref/templates/builtins/#date
func FormatDate(format string, date *time.Time) string {
	d := NowFunc()
	if date != nil {
		d = *date
	}

	_, weekNumber := d.ISOWeek()
	hour12NoZeros := d.Hour() % twelveHour
	if hour12NoZeros == 0 {
		hour12NoZeros = twelveHour
	}
	day := d.Day()
	weekDay := d.Weekday()
	year := d.Year()
	month := d.Month()
	hour := d.Hour()
	minute := d.Minute()
	second := d.Second()

	microSecond := d.Nanosecond() / thousand

	daysInCurrentMonth := func(t time.Time) int {
		firstDayNextMonth := time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, t.Location())
		lastDayCurrentMonth := firstDayNextMonth.AddDate(0, 0, -1)

		return lastDayCurrentMonth.Day()
	}

	replacer := strings.NewReplacer(
		"%d", fmt.Sprintf("%02d", day), // Day of the month, 2 digits with leading zeros. 01 to 31
		"%j", fmt.Sprintf("%01d", day), // Day of the month without leading zeros. 1 to 31
		"%D", weekDay.String()[:3], // Day of the week, textual, 3 letters. Fri
		"%l", weekDay.String(), // Day of the week, textual, long. Friday
		"%w", fmt.Sprintf("%d", weekDay), // Day of the week, digits without leading zeros. 0 (Sunday)
		"%z", strconv.Itoa(d.YearDay()), // Day of the year. 1 to 366
		"%W", strconv.Itoa(weekNumber), // ISO-8601 week number of year, with weeks starting on Monday. 1, 53

		"%m", fmt.Sprintf("%02d", month), // Month, 2 digits with leading zeros. 01 to 12
		"%n", fmt.Sprintf("%d", month), // Month without leading zeros. 1 to 12
		"%M", month.String()[:3], // Month, textual, 3 letters. Jan
		"%b", strings.ToLower(month.String()[:3]), // Month, textual, 3 letters, lowercase. jan
		"%F", month.String(), // Month, textual, long. January
		"%t", strconv.Itoa(daysInCurrentMonth(d)), // Number of days in the given month. 28 to 31

		"%y", fmt.Sprintf("%02d", year%modHundred), // Year, 2 digits with leading zeros. 00 to 99
		"%Y", fmt.Sprintf("%04d", year), // Year, 4 digits with leading zeros. 0001 ... 9999

		"%g", strconv.Itoa(hour12NoZeros), // Hour, 12-hour format without leading zeros. 1 to 12
		"%G", strconv.Itoa(hour), // Hour, 24-hour format without leading zeros. 0 to 23
		"%h", fmt.Sprintf("%02d", hour12NoZeros), // Hour, 12-hour format. 01 to 12
		"%H", fmt.Sprintf("%02d", hour), // Hour, 24-hour format. 00 to 23
		"%i", fmt.Sprintf("%02d", minute), // Minutes 00 to 59
		"%s", fmt.Sprintf("%02d", second), // Seconds, 2 digits with leading zeros. 00 to 59
		"%u", fmt.Sprintf("%06d", microSecond), // Microseconds. 000000 to 999999

		"%A", d.Format("PM"), // 'AM' or 'PM'.
	)

	return replacer.Replace(format)
}

func sanitizeFilename(input string) string {
	return illegalChars().ReplaceAllString(input, "_")
}

// GetFormattedFilename returns formated filename.
func GetFormattedFilename(s string, req *http.Request) string {
	if s == "" {
		return ""
	}

	u := req.URL.String()
	decoded, _ := url.QueryUnescape(u)

	argReplacer := strings.NewReplacer(
		"{hostname}", sanitizeFilename(req.Host),
		"{url}", sanitizeFilename(decoded),
	)
	sArgs := argReplacer.Replace(s)

	if strings.HasPrefix(sArgs, "~") {
		home, err := UserHomeDirFunc()
		if err != nil {
			sArgs = sArgs[1:]

			goto RETURN
		}

		sArgs = filepath.Join(home, sArgs[1:])
	}
RETURN:
	return filepath.Clean(FormatDate(sArgs, nil))
}
