package network

// Network configuration constants
const (
	// Bridge configuration
	BridgeName = "walkio-br0"
	BridgeIP   = "172.16.0.1"
	BridgeCIDR = "172.16.0.0/24"
	SubnetMask = "255.255.255.0"

	// IP pool configuration
	IPPoolStart = "172.16.0.2"
	IPPoolEnd   = "172.16.0.254"

	// Host port pool configuration
	HostPortPoolStart = 40000
	HostPortPoolEnd   = 50000

	// MAC address configuration
	MACPrefix = "AA:FC:00" // Locally administered, Firecracker hint

	// Default network settings for VMs
	DefaultGateway = BridgeIP
	DefaultDNS     = BridgeIP

	// TAP device naming
	TAPPrefix = "walkio-" // TAP devices: walkio-{last4timestamp}{last4uuid}
)

// NetworkConfig represents the complete network configuration for a VM.
// This is populated during VM creation and attached to VMInstance.
// If VMInstance.Network is nil, networking is disabled for that VM.
type NetworkConfig struct {
	VMID        string
	PortMapping []PortMapping
	TAPDevice   string // TAP device name (e.g., "walkio-7d3f89ab")
	IPAddress   string // Assigned IP address (e.g., "172.16.0.2")
	MACAddress  string // Generated MAC address (e.g., "AA:FC:00:A1:B2:C3")
	Gateway     string // Gateway IP (typically BridgeIP)
	DNS         string // DNS server IP (typically BridgeIP)
}

// PortMapping represents a TCP port forward from host to VM.
type PortMapping struct {
	HostPort  int
	GuestPort int
	Protocol  string
}
