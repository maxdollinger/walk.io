package network

import (
	"fmt"

	"github.com/vishvananda/netlink"
)

// GenerateTAPName creates a TAP device name from VM ID (UUID v7).
// Format: walkio-{last4timestamp}{last4uuid}
//
// UUID v7 structure (32 hex chars, no hyphens):
// - Chars 0-14: timestamp component (15 chars)
// - Chars 15-31: random component (17 chars)
//
// We extract:
// - Chars 11-14: last 4 of timestamp (4 chars)
// - Chars 28-31: last 4 of UUID (4 chars)
// Total: walkio- (7) + 8 hex chars = 15 chars (within Linux 15 char limit)
func GenerateTAPName(vmID string) string {
	// Ensure vmID is at least 32 characters (UUID v7 without hyphens)
	if len(vmID) < 32 {
		// Fallback for non-UUID IDs (shouldn't happen in normal operation)
		// Just take last 8 chars
		if len(vmID) >= 8 {
			return TAPPrefix + vmID[len(vmID)-8:]
		}
		return TAPPrefix + vmID
	}

	// Extract last 4 of timestamp (chars 11-14) and last 4 of UUID (chars 28-31)
	last4Timestamp := vmID[11:15]
	last4UUID := vmID[28:32]

	return TAPPrefix + last4Timestamp + last4UUID
}

// CreateTAP creates a TAP device and attaches it to the bridge.
// Returns the TAP device name.
func CreateTAP(vmID string) (string, error) {
	tapName := GenerateTAPName(vmID)

	// Check if TAP already exists
	if TAPExists(tapName) {
		return "", fmt.Errorf("%w: %s", ErrTAPNameExists, tapName)
	}

	// Create TAP device
	la := netlink.NewLinkAttrs()
	la.Name = tapName
	tap := &netlink.Tuntap{
		LinkAttrs: la,
		Mode:      netlink.TUNTAP_MODE_TAP,
	}

	if err := netlink.LinkAdd(tap); err != nil {
		return "", fmt.Errorf("%w: %v", ErrTAPCreateFailed, err)
	}

	// Get the bridge
	bridge, err := netlink.LinkByName(BridgeName)
	if err != nil {
		// Cleanup TAP device if we can't find bridge
		_ = netlink.LinkDel(tap)
		return "", fmt.Errorf("%w: %v", ErrBridgeNotFound, err)
	}

	// Attach TAP to bridge
	if err := netlink.LinkSetMaster(tap, bridge); err != nil {
		// Cleanup TAP device if we can't attach to bridge
		_ = netlink.LinkDel(tap)
		return "", fmt.Errorf("failed to attach TAP to bridge: %w", err)
	}

	// Bring TAP device up
	if err := netlink.LinkSetUp(tap); err != nil {
		// Cleanup TAP device if we can't bring it up
		_ = netlink.LinkDel(tap)
		return "", fmt.Errorf("failed to bring TAP up: %w", err)
	}

	return tapName, nil
}

// DestroyTAP removes a TAP device.
func DestroyTAP(name string) error {
	link, err := netlink.LinkByName(name)
	if err != nil {
		// TAP doesn't exist, nothing to do
		return nil
	}

	// Verify it's actually a TAP device
	if _, ok := link.(*netlink.Tuntap); !ok {
		return fmt.Errorf("device %s exists but is not a TAP device", name)
	}

	if err := netlink.LinkDel(link); err != nil {
		return fmt.Errorf("failed to delete TAP device %s: %w", name, err)
	}

	return nil
}

// TAPExists checks if a TAP device with the given name exists.
func TAPExists(name string) bool {
	link, err := netlink.LinkByName(name)
	if err != nil {
		return false
	}

	_, ok := link.(*netlink.Tuntap)
	return ok
}
