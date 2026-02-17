package network

import (
	"crypto/sha256"
	"fmt"
)

// GenerateMACAddress creates a MAC address from VM ID.
// Format: AA:FC:00:XX:XX:XX (last 3 octets from vmID hash)
//
// The prefix AA:FC:00 is:
// - AA: Locally administered (bit 1 set in first octet)
// - FC: Firecracker hint
// - 00: Reserved for extension
//
// The last 3 octets are derived from the VM ID to ensure uniqueness.
func GenerateMACAddress(vmID string) string {
	// Hash the VM ID to get deterministic but unique bytes
	hash := sha256.Sum256([]byte(vmID))

	// Use first 3 bytes of hash for last 3 octets
	return fmt.Sprintf("%s:%02X:%02X:%02X",
		MACPrefix,
		hash[0],
		hash[1],
		hash[2],
	)
}
