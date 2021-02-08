package go_tuntap

import "C"
import (
	"io"
	"syscall"
)

type LinuxVirtualNetworkInterface struct {
	mode        VirtualNetworkInterfaceMode
	name        string
	cStringName *C.char
	persistent  bool

	fd                 int
	mtu                VirtualNetworkInterfaceMTU
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
		mode:        mode,
		name:        name,
		cStringName: goString2CString(name),
		persistent:  persistent,

		fd:      fd,
	}

	/*
	if err := syscall.SetNonblock(vni.fd, true); err != nil {
		vni.Close()
		return nil, err
	}
	*/

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

func (l *LinuxVirtualNetworkInterface) Read(p []byte) (n int, err error) {
	n, err = syscall.Read(l.fd, p)
	if n == 0 && err == nil {
		return n, io.EOF
	}
	if err != nil {
		return 0, err
	}
	return
}

func (l *LinuxVirtualNetworkInterface) Write(p []byte) (n int, err error) {
	if len(p) > 0 {
		n, err = syscall.Write(l.fd, p)
		if n == 0 && err == nil {
			return n, io.EOF
		}
		if err != nil {
			return 0, err
		}
	}
	return
}

func (l *LinuxVirtualNetworkInterface) Close() error {
	if l.fd >= 0 {
		fd2Close := l.fd
		l.fd = -1
		err := syscall.Close(fd2Close)
		if err != nil {
			return err
		}
	}

	return nil
}

func createTun(name string) (fd int, err error) {
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
