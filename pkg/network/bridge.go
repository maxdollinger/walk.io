package network

import (
	"fmt"
	"net"

	"github.com/vishvananda/netlink"
)

// EnsureBridge creates the walkio bridge if it doesn't exist and configures its IP address.
// This is idempotent - safe to call multiple times.
func EnsureBridge() error {
	// Check if bridge already exists
	bridge, ok := GetWalkioBridge()
	if !ok {
		// Bridge doesn't exist, create it
		la := netlink.NewLinkAttrs()
		la.Name = BridgeName
		bridge = &netlink.Bridge{LinkAttrs: la}

		if err := netlink.LinkAdd(bridge); err != nil {
			return fmt.Errorf("%w: %v", ErrBridgeCreateFailed, err)
		}
	}

	// Ensure it's up and has correct IP
	return configureBridge(bridge)
}

// configureBridge sets the IP address and brings the bridge up.
func configureBridge(bridge *netlink.Bridge) error {
	// Parse and add IP address
	addr, err := netlink.ParseAddr(BridgeIP + "/24")
	if err != nil {
		return fmt.Errorf("failed to parse bridge IP: %w", err)
	}

	// Check if address is already assigned
	addrs, err := netlink.AddrList(bridge, netlink.FAMILY_V4)
	if err != nil {
		return fmt.Errorf("failed to list bridge addresses: %w", err)
	}

	hasIP := false
	for _, a := range addrs {
		if a.IP.Equal(addr.IP) {
			hasIP = true
			break
		}
	}

	// Add IP if not present
	if !hasIP {
		if err := netlink.AddrReplace(bridge, addr); err != nil {
			return fmt.Errorf("failed to add IP to bridge: %w", err)
		}
	}

	// Bring the bridge up
	if err := netlink.LinkSetUp(bridge); err != nil {
		return fmt.Errorf("failed to bring bridge up: %w", err)
	}

	return nil
}

// GetWalkioBridge checks if the walkio bridge exists.
func GetWalkioBridge() (*netlink.Bridge, bool) {
	link, err := netlink.LinkByName(BridgeName)
	if err != nil {
		return nil, false
	}

	bridge, ok := link.(*netlink.Bridge)
	if !ok {
		return nil, false
	}

	return bridge, ok
}

// DestroyBridge removes the walkio bridge.
// This will fail if any TAP devices are still attached.
func DestroyBridge() error {
	bridge, ok := GetWalkioBridge()
	if !ok {
		return nil
	}

	if err := netlink.LinkDel(bridge); err != nil {
		return fmt.Errorf("failed to delete bridge: %w", err)
	}

	return nil
}

// GetBridgeIP returns the bridge IP address as a net.IP.
func GetBridgeIP() net.IP {
	return net.ParseIP(BridgeIP)
}
