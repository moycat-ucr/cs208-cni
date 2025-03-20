package main

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"net"
	"sort"
)

func ParseNodeIP(node *corev1.Node) net.IP {
	return net.ParseIP(node.Status.Addresses[0].Address)
}

func ParseNodePodCIDRs(node *corev1.Node) ([]*net.IPNet, error) {
	cidrStrings := make(map[string]bool, len(node.Spec.PodCIDRs)+1)
	cidrStrings[node.Spec.PodCIDR] = true
	for _, cidr := range node.Spec.PodCIDRs {
		cidrStrings[cidr] = true
	}
	podCIDRs := make([]*net.IPNet, 0, len(cidrStrings))
	for cidr := range cidrStrings {
		if len(cidr) == 0 {
			continue
		}
		_, podCIDR, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, fmt.Errorf("cannot parse cidr [%s]: %w", cidr, err)
		}
		podCIDRs = append(podCIDRs, podCIDR)
	}
	sort.Slice(podCIDRs, func(i, j int) bool {
		return podCIDRs[i].String() < podCIDRs[j].String()
	})
	return podCIDRs, nil
}

func ParseIPNets(ipNetStrings []string) ([]*net.IPNet, error) {
	var ipNets []*net.IPNet
	for _, ipNetString := range ipNetStrings {
		_, ipNet, err := net.ParseCIDR(ipNetString)
		if err != nil {
			return nil, fmt.Errorf("cannot parse [%s]: %w", ipNetString, err)
		}
		ipNets = append(ipNets, ipNet)
	}
	sort.Slice(ipNets, func(i, j int) bool {
		return ipNets[i].String() < ipNets[j].String()
	})
	return ipNets, nil
}
