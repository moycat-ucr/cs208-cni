package main

import (
	"fmt"
	"github.com/coreos/go-iptables/iptables"
	"log"
	"net"
)

func initNAT(clusterPodCIDRs []*net.IPNet) error {
	tables, _ := iptables.NewWithProtocol(iptables.ProtocolIPv4)
	addRules := func(subnets []string) error {
		exists, err := tables.ChainExists(tableName, chainName)
		if err != nil {
			return fmt.Errorf("cannot create a unique chain: %w", err)
		}
		if !exists {
			if err := tables.NewChain(tableName, chainName); err != nil {
				return fmt.Errorf("cannot create a unique chain: %w", err)
			}
		}
		if err := tables.AppendUnique(tableName, chainName, "-p", "tcp", "-m", "tcp", "--tcp-flags", "SYN,RST", "SYN",
			"-j", "TCPMSS", "--clamp-mss-to-pmtu"); err != nil {
			return fmt.Errorf("cannot enable tcp mss clamping: %w", err)
		}
		if err := tables.AppendUnique(tableName, chainName, "-j", "MASQUERADE"); err != nil {
			return fmt.Errorf("cannot append the nat rule: %w", err)
		}
		for _, subnet := range subnets {
			log.Printf("adding nat rules for [%s]", subnet)
			// NAT if traffic comes from the subnet.
			if err := tables.AppendUnique(tableName, "POSTROUTING", "--src", subnet, "-j", chainName); err != nil {
				return fmt.Errorf("cannot redirect outgoing traffic from [%s]: %w", subnet, err)
			}
			// However, skip if traffic goes to the subnet.
			if err := tables.InsertUnique(tableName, chainName, 1, "--dst", subnet, "-j", "RETURN"); err != nil {
				return fmt.Errorf("cannot add nat exclusion rule for [%s]: %w", subnet, err)
			}
		}
		return nil
	}
	var subnets []string
	for _, cidr := range clusterPodCIDRs {
		subnets = append(subnets, cidr.String())
	}
	if err := addRules(subnets); err != nil {
		return fmt.Errorf("cannot setup nat for v4 subnets: %w", err)
	}
	log.Print("ipv4 nat rules are ready")
	return nil
}
