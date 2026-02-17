package network

import (
	"fmt"
	"os"
	"strconv"

	"github.com/coreos/go-iptables/iptables"
)

// EnableNAT sets up IP forwarding and MASQUERADE for internet access.
// This enables VMs to access the internet via the host.
func EnableNAT() error {
	// Enable IP forwarding
	if err := enableIPForwarding(); err != nil {
		return fmt.Errorf("failed to enable IP forwarding: %w", err)
	}

	// Create iptables instance
	ipt, err := iptables.New()
	if err != nil {
		return fmt.Errorf("failed to initialize iptables: %w", err)
	}

	// Add MASQUERADE rule for outbound traffic from VM network
	// iptables -t nat -A POSTROUTING -s 172.16.0.0/24 -j MASQUERADE
	err = ipt.AppendUnique("nat", "POSTROUTING", "-s", BridgeCIDR, "-j", "MASQUERADE")
	if err != nil {
		return fmt.Errorf("%w: failed to add MASQUERADE rule: %v", ErrNATSetupFailed, err)
	}

	// Add FORWARD rules to allow traffic through the bridge
	// iptables -A FORWARD -i walkio-br0 -j ACCEPT
	err = ipt.AppendUnique("filter", "FORWARD", "-i", BridgeName, "-j", "ACCEPT")
	if err != nil {
		return fmt.Errorf("%w: failed to add FORWARD rule: %v", ErrNATSetupFailed, err)
	}

	// iptables -A FORWARD -o walkio-br0 -j ACCEPT
	err = ipt.AppendUnique("filter", "FORWARD", "-o", BridgeName, "-j", "ACCEPT")
	if err != nil {
		return fmt.Errorf("%w: failed to add FORWARD rule: %v", ErrNATSetupFailed, err)
	}

	return nil
}

// DisableNAT removes NAT rules (cleanup).
func DisableNAT() error {
	ipt, err := iptables.New()
	if err != nil {
		return fmt.Errorf("failed to initialize iptables: %w", err)
	}

	// Remove MASQUERADE rule
	_ = ipt.Delete("nat", "POSTROUTING", "-s", BridgeCIDR, "-j", "MASQUERADE")

	// Remove FORWARD rules
	_ = ipt.Delete("filter", "FORWARD", "-i", BridgeName, "-j", "ACCEPT")
	_ = ipt.Delete("filter", "FORWARD", "-o", BridgeName, "-j", "ACCEPT")

	// Note: We don't disable IP forwarding as other services might be using it

	return nil
}

// AddPortMappings creates DNAT rules for port forwarding (batch operation).
// Maps host ports to VM guest ports.
func AddPortMappings(vmIP string, mappings []PortMapping) error {
	if len(mappings) == 0 {
		return nil
	}

	ipt, err := iptables.New()
	if err != nil {
		return fmt.Errorf("failed to initialize iptables: %w", err)
	}

	for _, mapping := range mappings {
		// Only support TCP for POC
		if mapping.Protocol != "tcp" {
			continue
		}

		// iptables -t nat -A PREROUTING -p tcp --dport {hostPort} -j DNAT --to-destination {vmIP}:{guestPort}
		err = ipt.AppendUnique("nat", "PREROUTING",
			"-p", "tcp",
			"--dport", strconv.Itoa(mapping.HostPort),
			"-j", "DNAT",
			"--to-destination", fmt.Sprintf("%s:%d", vmIP, mapping.GuestPort))

		if err != nil {
			return fmt.Errorf("failed to add port mapping %d->%s:%d: %w",
				mapping.HostPort, vmIP, mapping.GuestPort, err)
		}
	}

	return nil
}

// RemovePortMappings removes DNAT rules (batch operation).
func RemovePortMappings(vmIP string, mappings []PortMapping) error {
	if len(mappings) == 0 {
		return nil
	}

	ipt, err := iptables.New()
	if err != nil {
		return fmt.Errorf("failed to initialize iptables: %w", err)
	}

	for _, mapping := range mappings {
		// Only TCP for POC
		if mapping.Protocol != "tcp" {
			continue
		}

		// iptables -t nat -D PREROUTING -p tcp --dport {hostPort} -j DNAT --to-destination {vmIP}:{guestPort}
		_ = ipt.Delete("nat", "PREROUTING",
			"-p", "tcp",
			"--dport", strconv.Itoa(mapping.HostPort),
			"-j", "DNAT",
			"--to-destination", fmt.Sprintf("%s:%d", vmIP, mapping.GuestPort))
	}

	return nil
}

// SetupDNSRedirect redirects DNS queries from VMs to the host's DNS server.
// This is a simple redirect approach for POC.
func SetupDNSRedirect() error {
	// Read host's DNS server from /etc/resolv.conf
	// For POC, we'll just redirect to 8.8.8.8 (Google DNS)
	// In production, you'd parse /etc/resolv.conf to get the actual nameserver

	ipt, err := iptables.New()
	if err != nil {
		return fmt.Errorf("failed to initialize iptables: %w", err)
	}

	// For now, simple approach: let VMs use the bridge IP as DNS
	// and the host will handle forwarding
	// This requires dnsmasq or similar to be running on the bridge IP
	// For POC, we'll skip this and rely on VMs using 8.8.8.8 directly via NAT

	// Future: Add DNS forwarding rules here
	// iptables -t nat -A PREROUTING -d 172.16.0.1 -p udp --dport 53 -j DNAT --to-destination {hostDNS}
	// iptables -t nat -A PREROUTING -d 172.16.0.1 -p tcp --dport 53 -j DNAT --to-destination {hostDNS}

	_ = ipt // Suppress unused variable warning for now

	return nil
}

// enableIPForwarding enables IPv4 forwarding in the kernel.
func enableIPForwarding() error {
	const ipForwardPath = "/proc/sys/net/ipv4/ip_forward"

	// Check current value
	data, err := os.ReadFile(ipForwardPath)
	if err != nil {
		return fmt.Errorf("failed to read ip_forward: %w", err)
	}

	// Already enabled
	if len(data) > 0 && data[0] == '1' {
		return nil
	}

	// Enable it
	err = os.WriteFile(ipForwardPath, []byte("1"), 0644)
	if err != nil {
		return fmt.Errorf("%w: failed to write ip_forward: %v", ErrForwardingDisabled, err)
	}

	return nil
}
