package main

import "net"

const (
	tunnelPrefix  = "cni."
	cniConfigPath = "/etc/cni/net.d/208-cni.conflist"
	tableName     = "nat"
	chainName     = "CNI"
)

type KubeadmClusterConfiguration struct {
	Networking struct {
		PodSubnet string `yaml:"podSubnet"`
	} `yaml:"networking"`
}

type Nodes map[string]*Node

type Node struct {
	Name     string
	IP       net.IP
	PodCIDRs []*net.IPNet
	Tunnel   string
}

func (n *Node) HasUpdates(nn *Node) bool {
	if n == nil && nn == nil {
		return false
	}
	if (n == nil) != (nn == nil) {
		return true
	}
	if n.Name != nn.Name {
		return true
	}
	if !n.IP.Equal(nn.IP) {
		return true
	}
	if len(n.PodCIDRs) != len(nn.PodCIDRs) {
		return true
	}
	for i := range n.PodCIDRs {
		if n.PodCIDRs[i].String() != nn.PodCIDRs[i].String() {
			return true
		}
	}
	return false
}
