package go_tuntap

type VirtualNetworkInterfaceMode uint8
const (
	TUN VirtualNetworkInterfaceMode = iota
	TAP
)

type VirtualNetworkInterfaceState uint8
const (
	UP VirtualNetworkInterfaceState = iota
	DOWN
)

type VirtualNetworkInterfaceMTU int32

type VirtualNetworkInterface interface {
	GetMode() VirtualNetworkInterfaceMode
	GetName() string
	IsPersistent() bool
	SetFlags(flags int) error
	SetMTU(VirtualNetworkInterfaceMTU) error
	GetMTU() VirtualNetworkInterfaceMTU

	SetAddress(string) error

	Close()
}

type VirtualNetworkTUN interface {
	VirtualNetworkInterface
	SetDestinationAddress(address string) error
}