package validateutils_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vbyazilim/basichttpdebugger/internal/validateutils"
)

func TestValidateNetworkAddress(t *testing.T) {
	t.Run("Valid addresses", func(t *testing.T) {
		validAddresses := []string{
			":8080",
			":9002",
			"localhost:8080",
			"127.0.0.1:8080",
			"0.0.0.0:9002",
			"192.168.1.1:3000",
			":80",
			":443",
		}

		for _, addr := range validAddresses {
			err := validateutils.ValidateNetworkAddress(addr)
			assert.NoError(t, err, "expected %q to be valid", addr)
		}
	})

	t.Run("Invalid addresses", func(t *testing.T) {
		invalidAddresses := []string{
			"8080",
			"localhost",
			"invalid:port",
			":abc",
			":-1",
			":99999",
		}

		for _, addr := range invalidAddresses {
			err := validateutils.ValidateNetworkAddress(addr)
			assert.Error(t, err, "expected %q to be invalid", addr)
		}
	})
}
