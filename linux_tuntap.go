package go_tuntap

import "C"
import (
	"io"
	"os"
	"syscall"
)

type LinuxVirtualNetworkInterface struct {
	mode VirtualNetworkInterfaceMode
	name string
	cStringName *C.char
	persistent bool

	fd int
	mtu VirtualNetworkInterfaceMTU
	io.ReadWriteCloser
}

func NewLinuxVirtualNetworkInterface(mode VirtualNetworkInterfaceMode, name string, persistent bool) (*LinuxVirtualNetworkInterface, error) {
	var fd int
	var err error
	if mode == TUN {
		fd, err = createTun(name)
	} else {
		fd, err = createTap(name)
	}

	if err != nil {
		return nil, err
	}

	if persistent {
		err = ioctl(uintptr(fd), syscall.TUNSETPERSIST, 1)
		if err != nil {
			_ = syscall.Close(fd)
			return nil, err
		}
	}

	vni := &LinuxVirtualNetworkInterface{
		mode: mode,
		name: name,
		cStringName: goString2CString(name),
		persistent: persistent,

		fd: fd,
		mtu: 1500,
		ReadWriteCloser: os.NewFile(uintptr(fd), name),
	}

	if vni.GetMode() == TUN {
		err = vni.tunInit()
		if err != nil {
			vni.Close()
			return nil, err
		}
	} else {
		err = vni.SetFlags(syscall.IFF_UP)
	}

	return vni, nil
}

func (l *LinuxVirtualNetworkInterface) GetMode() VirtualNetworkInterfaceMode {
	return l.mode
}

func (l *LinuxVirtualNetworkInterface) GetName() string {
	return l.name
}

func (l *LinuxVirtualNetworkInterface) IsPersistent() bool {
	return l.persistent
}

func (l *LinuxVirtualNetworkInterface) SetFlags(flags int) error {
	return setFlags(l.cStringName, flags)
}

func (l *LinuxVirtualNetworkInterface) SetMTU(mtu VirtualNetworkInterfaceMTU) error {
	if err := setMTU(l.cStringName, int(mtu)); err != nil {
		l.mtu = mtu
	}
	return nil
}

func (l *LinuxVirtualNetworkInterface) GetMTU() VirtualNetworkInterfaceMTU {
	return l.mtu
}

func (l *LinuxVirtualNetworkInterface) SetAddress(address string, netmask string) error {
	return setAddress(l.cStringName, address, netmask)
}

func (l *LinuxVirtualNetworkInterface) SetBinaryAddress(address uint32, netmask uint32) error {
	return setUInt32Address(l.cStringName, address, netmask)
}

func (l *LinuxVirtualNetworkInterface) SetDestinationAddress(address string) error {
	return setTunDestinationAddress(l.cStringName, address)
}

func (l *LinuxVirtualNetworkInterface) SetBinaryDestinationAddress(address uint32) error {
	return setTunUInt32DestinationAddress(l.cStringName, address)
}

func createTun(name string) (fd int, err error)  {
	fd, err = vniAlloc(1, name)
	return
}

func createTap(name string) (fd int, err error) {
	fd, err = vniAlloc(2, name)
	return
}

func (l *LinuxVirtualNetworkInterface) tunInit() error {
	return tunInit(l.cStringName)
}