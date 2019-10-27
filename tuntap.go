package go_tuntap

import "io"

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

type VirtualNetworkInterfaceMTU uint16

type VirtualNetworkInterface interface {
	GetMode() VirtualNetworkInterfaceMode
	GetName() string
	IsPersistent() bool
	SetFlags(flags int) error
	SetMTU(VirtualNetworkInterfaceMTU) error
	GetMTU() VirtualNetworkInterfaceMTU

	SetAddress(string, string) error
	SetBinaryAddress(uint32, uint32) error

	io.ReadWriteCloser
}

type VirtualNetworkTUN interface {
	VirtualNetworkInterface
	SetDestinationAddress(address string) error
	SetBinaryDestinationAddress(uint32) error
}