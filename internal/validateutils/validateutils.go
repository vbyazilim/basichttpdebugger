package validateutils

import (
	"fmt"
	"net"
)

// ValidateNetworkAddress validates given network addr.
func ValidateNetworkAddress(adrr string) error {
	if _, err := net.ResolveTCPAddr("tcp", adrr); err != nil {
		return fmt.Errorf("invalid tcp addr: %w", err)
	}

	return nil
}
