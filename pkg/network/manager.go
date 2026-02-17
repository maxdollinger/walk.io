package network

// NetworkManager is the central coordinator for all networking operations.
// It manages IP allocation, TAP devices, port mappings, and ensures
// consistent state across all network resources.
//
// This should be created once at application startup and passed as a
// dependency to components that need networking functionality.
type NetworkManager struct {
	// Resource managers (each has its own mutex)
	ipPool       *IPPool
	hostPortPool *HostPortPool

	// Infrastructure state
	bridgeInitialized bool // Whether bridge and NAT are set up
}

// NewNetworkManager creates a new NetworkManager instance.
// This does not set up network infrastructure - call EnsureInfrastructure() separately.
func NewNetworkManager() (*NetworkManager, error) {
	portPool, err := NewHostPortPool(HostPortPoolStart, HostPortPoolEnd)
	if err != nil {
		return nil, err
	}

	return &NetworkManager{
		ipPool:            NewIPPool(),
		hostPortPool:      portPool,
		bridgeInitialized: false,
	}, nil
}
