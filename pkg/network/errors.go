package network

import "errors"

var (
	// IP Pool errors
	ErrIPPoolExhausted = errors.New("no available IP addresses in pool")
	ErrIPNotAllocated  = errors.New("IP address is not currently allocated")
	ErrIPAlreadyInUse  = errors.New("IP address is already in use")

	// Port pool errors
	ErrPortPoolExhausted = errors.New("no available ports in pool")

	// Port mapping errors
	ErrHostPortInUse   = errors.New("host port is already in use")
	ErrInvalidPort     = errors.New("invalid port number (must be 1-65535)")
	ErrMappingNotFound = errors.New("port mapping not found")

	// Bridge errors
	ErrBridgeNotFound     = errors.New("bridge device not found")
	ErrBridgeCreateFailed = errors.New("failed to create bridge device")

	// TAP device errors
	ErrTAPCreateFailed = errors.New("failed to create TAP device")
	ErrTAPNotFound     = errors.New("TAP device not found")
	ErrTAPNameExists   = errors.New("TAP device name already exists")

	// NAT/iptables errors
	ErrNATSetupFailed     = errors.New("failed to setup NAT rules")
	ErrForwardingDisabled = errors.New("IP forwarding is disabled")

	// Permission errors
	ErrNeedRoot = errors.New("operation requires root privileges")
)
