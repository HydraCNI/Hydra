package cni

import (
	"context"
	"crypto/sha512"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/containernetworking/cni/libcni"
)

// Protocol parameters are passed to the plugins via OS environment variables.
const (
	EnvCNIPath        = "CNI_PATH"
	EnvNetDir         = "NETCONFPATH"
	EnvCapabilityArgs = "CAP_ARGS"
	EnvCNIArgs        = "CNI_ARGS"
	EnvCNIIfname      = "CNI_IFNAME"

	DefaultNetDir = "/etc/cni/net.d"

	CmdAdd   = "ADD"
	CmdCheck = "CHECK"
	CmdDel   = "DEL"
)

func parseArgs(args string) ([][2]string, error) {
	var result [][2]string

	pairs := strings.Split(args, ";")
	for _, pair := range pairs {
		kv := strings.Split(pair, "=")
		if len(kv) != 2 || kv[0] == "" || kv[1] == "" {
			return nil, fmt.Errorf("invalid CNI_ARGS pair %q", pair)
		}

		result = append(result, [2]string{kv[0], kv[1]})
	}

	return result, nil
}

type CNIPlugin struct {
	Name string
}

var DefaultCNIPlugin CNIPlugin

func (c *CNIPlugin) AddNetworkInterface(nsPath string) error {
	return c.NetworkInterfaceOpt(nsPath, CmdAdd)
}
func (c *CNIPlugin) DelNetworkInterface(nsPath string) error {
	return c.NetworkInterfaceOpt(nsPath, CmdDel)

}
func (c *CNIPlugin) CheckNetworkInterface(nsPath string) error {
	return c.NetworkInterfaceOpt(nsPath, CmdCheck)
}

func (c *CNIPlugin) NetworkInterfaceOpt(nsPath string, cniOpt string) error {
	// get default net conf dir
	netdir := os.Getenv(EnvNetDir)
	if netdir == "" {
		netdir = DefaultNetDir
	}

	netconf, err := libcni.LoadConfList(netdir, c.Name)
	if err != nil {
		fmt.Printf("get cni configuration failed: %s", err)
		return err
	}

	// what's cap argus
	var capabilityArgs map[string]interface{}
	capabilityArgsValue := os.Getenv(EnvCapabilityArgs)
	if len(capabilityArgsValue) > 0 {
		if err = json.Unmarshal([]byte(capabilityArgsValue), &capabilityArgs); err != nil {
			return err
		}
	}

	var cniArgs [][2]string
	args := os.Getenv(EnvCNIArgs)
	if len(args) > 0 {
		cniArgs, err = parseArgs(args)
		if err != nil {
			return err
		}
	}

	ifName, ok := os.LookupEnv(EnvCNIIfname)
	if !ok {
		ifName = fmt.Sprintf("eth99")
	}

	netns, err := filepath.Abs(nsPath)
	if err != nil {
		return err
	}

	// Generate the containerd by hashing the netns path
	s := sha512.Sum512([]byte(netns))
	containerID := fmt.Sprintf("cnitool-%x", s[:10])
	cninet := libcni.NewCNIConfig(filepath.SplitList(os.Getenv(EnvCNIPath)), nil)
	rt := &libcni.RuntimeConf{
		ContainerID:    containerID,
		NetNS:          netns,
		IfName:         ifName,
		Args:           cniArgs,
		CapabilityArgs: capabilityArgs,
	}
	switch cniOpt {
	case CmdAdd:
		result, err := cninet.AddNetworkList(context.TODO(), netconf, rt)
		if result != nil {
			_ = result.Print()
		}
		return err
	case CmdCheck:
		err = cninet.CheckNetworkList(context.TODO(), netconf, rt)
		return err
	case CmdDel:
		err = cninet.DelNetworkList(context.TODO(), netconf, rt)
		return err
	}
	return err
}
