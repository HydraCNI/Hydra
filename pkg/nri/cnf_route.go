package nri

import (
	"net"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
	"golang.org/x/sys/unix"

	"github.com/hydra-cni/hydra/pkg/cni"
	"github.com/hydra-cni/hydra/pkg/kubeclient"
)

const (
	GlobalCIDR        = "GLOBALCIDR"
	defaultGlobalCIDR = "10.112.0.0/12"
)

var (
	globalCIDR string
	dstIP      *net.IPNet
)

func init() {
	globalCIDR = os.Getenv(GlobalCIDR)
	if globalCIDR == "" {
		globalCIDR = defaultGlobalCIDR
	}

	_, ipNet, err := net.ParseCIDR(globalCIDR)
	if err != nil {
		logrus.Errorf("parse cidr failed")
	}
	dstIP = ipNet
}

func addRouteToCNFPod(nsPath string) error {
	nsHandle, err := netns.GetFromPath(nsPath)
	if err != nil {
		logrus.Errorf("Error opening namespace: %v", err)
		return err
	}
	defer nsHandle.Close()

	nlHandle, err := netlink.NewHandleAt(nsHandle)
	if err != nil {
		logrus.Errorf("Error opening namespace: %v", err)
		return err
	}
	defer nlHandle.Delete()

	// get the dedicated cni interface, default is eth99
	link, err := nlHandle.LinkByName(cni.IfName)
	if err != nil {
		logrus.Errorf("get link failed: %v", err)
		return err
	}

	// bring the interface up
	if err := nlHandle.LinkSetUp(link); err != nil {
		logrus.Errorf("netlinke set up failed: %v", err)
		return err
	}

	// ------------------- add a route to CNF pod ------------------------------
	ips, err := nlHandle.AddrList(link, unix.AF_INET)
	logrus.Debugf("ips of hydra: %v ", ips)
	if err != nil {
		logrus.Errorf("n: %v", err)
		return err
	}
	ip := net.IP{}
	if len(ips) > 0 {
		ip = ips[0].IP
	}
	logrus.Debugf("ip of hydra >>>>> : %v ", ip)
	// get cnf Pod IP, work as gateway
	gateway, err := kubeclient.GetCNFPodDedicatedCNIIP()
	logrus.Debugf("the cnf IP is %v", gateway)
	if err != nil {
		return err
	}
	route := netlink.Route{LinkIndex: link.Attrs().Index, Gw: gateway, Dst: dstIP, Src: ip}
	if err := nlHandle.RouteAdd(&route); err != nil {
		logrus.Errorf("Error adding route: %v", err)
		return err
	}
	return nil
}
