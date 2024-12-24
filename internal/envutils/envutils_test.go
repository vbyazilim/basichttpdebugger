package envutils_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vbyazilim/basichttpdebugger/internal/envutils"
)

func TestGetenvOrDefault(t *testing.T) {
	os.Setenv("TEST_STRING", "hello")
	os.Setenv("TEST_BOOL_TRUE", "true")
	os.Setenv("TEST_BOOL_FALSE", "false")
	os.Setenv("TEST_BOOL_ILLEGAL", "hello")
	os.Setenv("TEST_INT_VAL", "999")
	os.Unsetenv("TEST_NON_EXISTENT")

	defer func() {
		os.Unsetenv("TEST_STRING")
		os.Unsetenv("TEST_BOOL_TRUE")
		os.Unsetenv("TEST_BOOL_FALSE")
		os.Unsetenv("TEST_BOOL_ILLEGAL")
		os.Unsetenv("TEST_INT_VAL")
	}()

	t.Run("String retrieval", func(t *testing.T) {
		val := envutils.GetenvOrDefault("TEST_STRING", "default")
		assert.Equal(t, "hello", val)

		val = envutils.GetenvOrDefault("TEST_NON_EXISTENT", "default")
		assert.Equal(t, "default", val)
	})

	t.Run("Boolean retrieval", func(t *testing.T) {
		val := envutils.GetenvOrDefault("TEST_BOOL_TRUE", false)
		assert.Equal(t, true, val)

		val = envutils.GetenvOrDefault("TEST_BOOL_FALSE", true)
		assert.Equal(t, false, val)

		val = envutils.GetenvOrDefault("TEST_NON_EXISTENT", true)
		assert.Equal(t, true, val)
	})

	t.Run("Fallback for non-existent variable", func(t *testing.T) {
		val := envutils.GetenvOrDefault("TEST_NON_EXISTENT", "fallback")
		assert.Equal(t, "fallback", val)
	})

	t.Run("Fallback for illegal boolean variable", func(t *testing.T) {
		val := envutils.GetenvOrDefault("TEST_BOOL_ILLEGAL", false)
		assert.Equal(t, false, val)
	})

	t.Run("Fallback for non matching case", func(t *testing.T) {
		val := envutils.GetenvOrDefault("TEST_INT_VAL", 0)
		assert.Equal(t, 0, val)
	})
}
