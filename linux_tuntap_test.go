package go_tuntap

import (
	"testing"
)

func TestLinuxVirtualNetworkInterface(t *testing.T) {
	var vni VirtualNetworkInterface
	var err error
	vni, err = NewLinuxVirtualNetworkInterface(TUN, "test_tun0", false)
	if err != nil {
		t.Errorf("NewLinuxVirtualNetworkInterface(): %s", err.Error())
		return
	}

	err = vni.SetMTU(VirtualNetworkInterfaceMTU(1300))
	if err != nil {
		t.Errorf("SetMTU(): %s", err.Error())
	}
	err = vni.SetAddress("1.2.3.1")
	if err != nil {
		t.Errorf("SetMTU(): %s", err.Error())
	}

	vniT := vni.(VirtualNetworkTUN)
	err = vniT.SetDestinationAddress("1.2.3.2")
	if err != nil {
		t.Errorf("SetMTU(): %s", err.Error())
	}
	vni.Close()
}