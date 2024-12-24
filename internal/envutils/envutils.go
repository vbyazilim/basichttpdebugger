//nolint:ireturn
package envutils

import (
	"os"
	"strconv"
)

// GetenvOrDefault checks the given environment variable.
// If it exists, it returns the value; otherwise, it returns the fallback value.
func GetenvOrDefault[T any](name string, fallback T) T {
	val := os.Getenv(name)
	if val == "" {
		return fallback
	}

	switch any(fallback).(type) {
	case string:
		if v, ok := any(val).(T); ok {
			return v
		}
	case bool:
		parsed, err := strconv.ParseBool(val)
		if err != nil {
			return fallback
		}

		if v, ok := any(parsed).(T); ok {
			return v
		}
	}

	return fallback
}
