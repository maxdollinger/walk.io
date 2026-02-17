package network

import (
	"fmt"
	"net"
	"sync"
)

// IPPool manages allocation of IP addresses from a defined pool.
// Thread-safe for concurrent VM creation.
type IPPool struct {
	mu   sync.RWMutex
	pool map[string]string // IP -> VMID mapping
}

// NewIPPool creates and initializes a new IP pool with the configured range.
// Parses IPPoolStart to IPPoolEnd and populates the available slice.

// Initialize populates the IP pool with available addresses from the configured range.
// This must be called before any allocations can be made.
func NewIPPool(ipPoolStart, ipPoolEnd string) (*IPPool, error) {
	startIP := net.ParseIP(ipPoolStart)
	endIP := net.ParseIP(ipPoolEnd)

	if startIP == nil || endIP == nil {
		return nil, fmt.Errorf("invalid IP pool range: start=%s, end=%s", IPPoolStart, IPPoolEnd)
	}

	// Convert IPs to 4-byte representation
	startIP = startIP.To4()
	endIP = endIP.To4()

	if startIP == nil || endIP == nil {
		return nil, fmt.Errorf("IP pool range must be IPv4 addresses")
	}

	// Convert to uint32 for easy iteration
	start := ipToUint32(startIP)
	end := ipToUint32(endIP)

	if start > end {
		return nil, fmt.Errorf("IP pool start (%s) is greater than end (%s)", IPPoolStart, IPPoolEnd)
	}

	pool := make(map[string]string, end-start)
	for i := start; i <= end; i++ {
		ip := uint32ToIP(i)
		pool[ip.String()] = ""
	}

	return &IPPool{pool: pool}, nil
}

// AllocateIP assigns a random IP address to a VM.
// Returns the allocated IP or an error if the pool is exhausted.
func (p *IPPool) AllocateIP(vmID string) (net.IP, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	var allocatedIP string
	for ip, allocatedVM := range p.pool {
		if allocatedVM == "" {
			p.pool[ip] = vmID
			allocatedIP = ip
			break
		}
	}

	if len(allocatedIP) == 0 {
		return nil, ErrIPNotAllocated
	}

	return net.ParseIP(allocatedIP), nil
}

// ReleaseIP returns an IP address back to the available pool.
// Returns an error if the IP is not currently allocated to the specified VM.
func (p *IPPool) ReleaseIP(ip *net.IP, vmID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	allocatedVM, exists := p.pool[ip.String()]
	if !exists {
		return ErrIPNotAllocated
	}

	if allocatedVM != vmID {
		return fmt.Errorf("IP %s is allocated to VM %s, not %s", ip, allocatedVM, vmID)
	}

	// Remove from allocated
	p.pool[ip.String()] = ""

	return nil
}

// IsAllocated checks if an IP address is currently allocated.
func (p *IPPool) IsAllocated(ip *net.IP) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	allocatedVM, exists := p.pool[ip.String()]
	return exists && allocatedVM != ""
}

// Helper functions for IP address arithmetic
func ipToUint32(ip net.IP) uint32 {
	ip = ip.To4()
	return uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
}

func uint32ToIP(n uint32) net.IP {
	return net.IPv4(byte(n>>24), byte(n>>16), byte(n>>8), byte(n))
}
