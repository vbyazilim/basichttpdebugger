package stringutils

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	twelveHour = 12
	thousand   = 1000
	modHundred = 100
)

// FormatDate formats given date as Django's format date style (most of them).
// https://docs.djangoproject.com/en/5.1/ref/templates/builtins/#date
func FormatDate(format string, date *time.Time) string {
	d := time.Now()
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
		"%d", fmt.Sprintf("%02d", day),
		"%j", fmt.Sprintf("%01d", day),
		"%D", weekDay.String()[:3],
		"%l", weekDay.String(),
		"%w", fmt.Sprintf("%d", weekDay),
		"%z", strconv.Itoa(d.YearDay()),
		"%W", strconv.Itoa(weekNumber),

		"%m", fmt.Sprintf("%02d", month),
		"%n", fmt.Sprintf("%d", month),
		"%M", month.String()[:3],
		"%b", strings.ToLower(month.String()[:3]),
		"%F", month.String(),
		"%t", strconv.Itoa(daysInCurrentMonth(d)),

		"%y", fmt.Sprintf("%02d", year%modHundred),
		"%Y", fmt.Sprintf("%04d", year),

		"%g", strconv.Itoa(hour12NoZeros),
		"%G", strconv.Itoa(hour),
		"%h", fmt.Sprintf("%02d", hour12NoZeros),
		"%H", fmt.Sprintf("%02d", hour),
		"%i", fmt.Sprintf("%02d", minute),
		"%s", fmt.Sprintf("%02d", second),
		"%u", fmt.Sprintf("%06d", microSecond),

		"%A", d.Format("PM"),
	)

	return replacer.Replace(format)
}
