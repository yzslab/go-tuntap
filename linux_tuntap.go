package go_tuntap

/*
#include <sys/eventfd.h>
*/
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
	epfd               int
	eventFD            int
	canRead            chan struct{}
	wouldBlock         chan struct{}
	stopEPollGoroutine chan struct{}
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
		epfd:    -1,
		eventFD: -1,
	}

	if err := syscall.SetNonblock(vni.fd, true); err != nil {
		vni.Close()
		return nil, err
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

	if err := vni.epollInit(); err != nil {
		vni.Close()
		return nil, err
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
	var ok bool
	if len(p) > 0 {
		goto StartRead // read till resource temporarily unavailable
	EPollWait:
		_, ok = <-l.canRead
		if ok == false {
			return 0, io.EOF
		}
	StartRead:
		n, err = syscall.Read(l.fd, p)
		if n == 0 && err == nil {
			return n, io.EOF
		}
		if err == syscall.EAGAIN || err == syscall.EWOULDBLOCK {
			l.wouldBlock <- struct{}{}
			goto EPollWait
		} else if err != nil {
			return 0, err
		}
	}
	return
}

func (l *LinuxVirtualNetworkInterface) Write(p []byte) (n int, err error) {
	if len(p) > 0 {
		n, err = syscall.Write(l.fd, p)
		if n == 0 && err == nil {
			return n, io.EOF
		}
		if err == syscall.EAGAIN || err == syscall.EWOULDBLOCK {
			return 0, nil // simply drop packets
		} else if err != nil {
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

	if l.eventFD >= 0 {
		syscall.Write(l.eventFD, []byte{0, 1, 2, 3, 4, 5, 6, 7})
		// if close eventfd here,
		// epoll_wait() can be wake up, by eventfd
		// so a dedicated goroutine for epoll_wait() is necessary for closing eventfd correctly
		l.eventFD = -1
	}

	if l.stopEPollGoroutine != nil {
		close(l.stopEPollGoroutine)
		l.stopEPollGoroutine = nil
	}
	return nil
}

func (l *LinuxVirtualNetworkInterface) readable() bool {
	return !(l.fd < 0 || l.epfd < 0 || l.eventFD < 0 || l.stopEPollGoroutine == nil)
}

func (l *LinuxVirtualNetworkInterface) epollInit() error {
	var err error
	l.epfd, err = syscall.EpollCreate1(0)
	if err != nil {
		return err
	}

	var event1 syscall.EpollEvent
	var event2 syscall.EpollEvent
	var eventFD int

	// add vni fd to epoll
	epollet := syscall.EPOLLET
	event1.Events = uint32(syscall.EPOLLIN) | uint32(epollet)
	event1.Fd = int32(l.fd)
	err = syscall.EpollCtl(l.epfd, syscall.EPOLL_CTL_ADD, l.fd, &event1)
	if err != nil {
		goto CloseEPoll
	}

	// create event fd
	eventFD = int(C.eventfd(0, C.EFD_CLOEXEC|C.EFD_NONBLOCK))
	if eventFD < 0 {
		goto CloseEPoll
	}
	l.eventFD = eventFD

	// add event fd to epoll
	event2.Events = syscall.EPOLLIN
	event2.Fd = int32(eventFD)
	err = syscall.EpollCtl(l.epfd, syscall.EPOLL_CTL_ADD, eventFD, &event2)
	if err != nil {
		goto CloseEventFD
	}

	l.canRead = make(chan struct{})
	l.wouldBlock = make(chan struct{})
	l.stopEPollGoroutine = make(chan struct{})
	go func() {
		defer func() {
			syscall.Close(eventFD)
			syscall.Close(l.epfd)
			close(l.canRead)
			close(l.wouldBlock)
		}()
		events := make([]syscall.EpollEvent, 2)

		for {
			select {
			case _, ok := <-l.wouldBlock:
				if ok == false {
					return
				}
				break
			case <-l.stopEPollGoroutine:
				return
			}

			n, err := syscall.EpollWait(l.epfd, events, -1)
			/*
				if n > 0 {
					for _, event := range events[:n] {
						if event.Fd == int32(eventFD) {
							log.Println("has event fd")
							return
						}
					}
				}
			*/

			// waken after vni close check
			if !l.readable() {
				return
			}

			if err == syscall.EINTR {
				continue // continue on interrupted
			} else if err != nil {
				return
			} else if n == 0 {
				continue
			}

			l.canRead <- struct{}{}
		}
	}()
	return nil

CloseEventFD:
	syscall.Close(l.eventFD)
	l.eventFD = -1
CloseEPoll:
	syscall.Close(l.epfd)
	l.epfd = -1

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
