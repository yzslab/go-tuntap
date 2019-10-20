package go_tuntap

/*
#cgo CFLAGS: -I${SRCDIR}/libs
#cgo LDFLAGS: -L${SRCDIR}/libs -l:libtuntap4go.a
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <netinet/in.h>
#include "tuntap4go.h"
*/
import "C"

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

func ioctl(fd uintptr, request uintptr, argp uintptr) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(request), argp)
	if errno != 0 {
		return os.NewSyscallError("ioctl", errno)
	}
	return nil
}

func getErrno() int {
	return int(C.get_errno());
}

func getErrorString() string {
	errorCString := C.get_strerror_r(C.get_errno())
	defer C.free(unsafe.Pointer(errorCString))
	return C.GoString(errorCString)
}

func vniAlloc(mode int16, name string) (int, error) {
	nameCString := C.CString(name)
	defer C.free(unsafe.Pointer(nameCString))
	fd := int(C.vni_alloc(C.short(mode), nameCString))
	if fd < 0 {
		return fd, fmt.Errorf("vniAlloc(): %s", getErrorString())
	}
	return fd, nil
}

func setFlags(nameCString *C.char, flags int) error {
	if ret := C.set_vni_flags(nameCString, C.int(flags)); ret < 0 {
		return fmt.Errorf("tun_init(): %s", getErrorString())
	}
	return nil
}

func tunInit(nameCString *C.char) error {
	returnValue := C.tun_init(nameCString)
	if returnValue < 0 {
		return fmt.Errorf("tun_init(): %s", getErrorString())
	}
	return nil
}

func setMTU(nameCString *C.char, mtu int) error {
	if ret := C.set_mtu(nameCString, C.int(mtu)); ret < 0 {
		return fmt.Errorf("set_mtu(): %s", getErrorString())
	}
	return nil
}

func setAddress(nameCString *C.char, address string) error {
	addressCString := goString2CString(address);
	defer freeCString(addressCString)
	if ret := C.set_vni_address_by_ascii(nameCString, addressCString); ret < 0 {
		return fmt.Errorf("set_vni_address_by_ascii(): %s", getErrorString())
	}
	return nil
}

func setTunDestinationAddress(nameCString *C.char, address string) error {
	addressCString := goString2CString(address);
	defer freeCString(addressCString)
	if ret := C.set_tun_destination_address_by_ascii(nameCString, addressCString); ret < 0 {
		return fmt.Errorf("set_tun_destination_address_by_ascii(): %s", getErrorString())
	}
	return nil
}

func goString2CString(goString string) *C.char {
	return C.CString(goString)
}

func cString2GoString(cString *C.char) string {
	return C.GoString(cString)
}

func freeCString(cString *C.char) {
	C.free(unsafe.Pointer(cString))
}