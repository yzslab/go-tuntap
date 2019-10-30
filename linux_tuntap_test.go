package go_tuntap

import (
	"goipam"
	"log"
	"testing"
	"time"
)

func TestLinuxVirtualNetworkInterface(t *testing.T) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	var vni VirtualNetworkInterface
	var err error
	vni, err = NewLinuxVirtualNetworkInterface(TAP, "test_tun0", false)
	if err != nil {
		t.Errorf("NewLinuxVirtualNetworkInterface(): %s", err.Error())
		return
	}

	err = vni.SetMTU(VirtualNetworkInterfaceMTU(1300))
	if err != nil {
		t.Errorf("SetMTU(): %s", err.Error())
	}
	err = vni.SetAddress("1.2.3.1", "255.255.255.255")
	if err != nil {
		t.Errorf("SetMTU(): %s", err.Error())
	}

	vniT := vni.(VirtualNetworkTUN)
	err = vniT.SetBinaryDestinationAddress(Htonl(0xfffffff0))
	if err != nil {
		t.Errorf("SetDestinationAddress(): %s", err.Error())
	}

	if 0xfffffff0 != Ntohl(Htonl(0xfffffff0)) {
		t.Errorf("%s", "ntohl or htonl have bug(s)")
	}

	go func() {
		for {
			buffer := make([]byte, 1522)
			_, err := vni.Read(buffer)
			if err != nil {
				log.Println(err)
				break
			}
		}
		log.Println("read goroutine end")
	}()

	time.Sleep(3 * time.Second)

	vni.Close()

	vni, err = NewLinuxVirtualNetworkInterface(TAP, "test_tap0", false)
	if err != nil {
		t.Errorf("NewLinuxVirtualNetworkInterface(): %s", err.Error())
		return
	}

	longIP, _ := goipam.IP2long("1.2.3.2")
	longNetmask, _ := goipam.IP2long("255.255.255.0")

	err = vni.SetBinaryAddress(Htonl(longIP), Htonl(longNetmask))
	if err != nil {
		t.Errorf("SetMTU(): %s", err.Error())
	}

	vni.Close()

	return
}
