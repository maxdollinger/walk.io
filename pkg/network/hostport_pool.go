package network

import (
	"fmt"
	"sync"
)

// HostPortPool manages allocation of host ports from a defined pool.
// Thread-safe for concurrent VM creation.
type HostPortPool struct {
	mu   sync.RWMutex
	pool map[int]string // port -> vmID mapping
}

// NewHostPortPool creates a new host port pool.
func NewHostPortPool(startPort int, endPort int) (*HostPortPool, error) {
	if HostPortPoolStart >= HostPortPoolEnd {
		return nil, fmt.Errorf("invalid port pool range: start=%d, end=%d", HostPortPoolStart, HostPortPoolEnd)
	}

	hostPortPool := &HostPortPool{
		pool: make(map[int]string),
	}

	for port := startPort; port <= endPort; port++ {
		hostPortPool.pool[port] = ""
	}

	return hostPortPool, nil
}

// AllocatePorts assigns N random ports to a VM.
// Returns the allocated ports or an error if not enough ports are available.
func (p *HostPortPool) AllocatePorts(vmID string, count int) ([]int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if count <= 0 {
		return []int{}, nil
	}

	ports := make([]int, 0, count)
	for port, id := range p.pool {
		if len(id) == 0 {
			ports = append(ports, port)
		}

		if len(ports) == count {
			break
		}
	}

	if len(ports) < count {
		return nil, ErrPortPoolExhausted
	}

	for _, port := range ports {
		p.pool[port] = vmID
	}

	return ports, nil
}

// ReleasePorts returns ports back to the available pool.
// Returns an error if any port is not currently allocated to the specified VM.
func (p *HostPortPool) ReleasePorts(ports []int, vmID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, port := range ports {

		// validate port allocation
		allocatedVM, ok := p.pool[port]
		if !ok {
			return fmt.Errorf("port %d is not int the pool", port)
		}

		if len(allocatedVM) > 0 && allocatedVM != vmID {
			return fmt.Errorf("port %d is allocated to VM %s, not %s", port, allocatedVM, vmID)
		}

		// release ports
		p.pool[port] = ""
	}

	return nil
}

// IsAllocated checks if a port is currently allocated.
func (p *HostPortPool) IsAllocated(port int) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	allocatedVM, ok := p.pool[port]
	return ok && len(allocatedVM) > 0
}
