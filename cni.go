package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
)

func initCNI(nodePodCIDRs []*net.IPNet) error {
	routes := []map[string]interface{}{{"dst": "0.0.0.0/0"}}
	podCIDRs := make([][]map[string]interface{}, 0, len(nodePodCIDRs))
	for _, cidr := range nodePodCIDRs {
		podCIDRs = append(podCIDRs, []map[string]interface{}{{"subnet": cidr.String()}})
	}
	config := map[string]interface{}{
		"name":       "cni",
		"cniVersion": "0.3.1",
		"plugins": []map[string]interface{}{
			{
				"type": "ptp",
				"ipam": map[string]interface{}{
					"type":   "host-local",
					"ranges": podCIDRs,
					"routes": routes,
				},
			},
			{
				"type":         "portmap",
				"snat":         true,
				"capabilities": map[string]bool{"portMappings": true},
			},
		},
	}
	b, _ := json.Marshal(config)
	if err := os.WriteFile(cniConfigPath, b, 0o644); err != nil {
		return fmt.Errorf("cannot write cni config [%s]: %w", cniConfigPath, err)
	}
	log.Printf("write cni config to [%s]", cniConfigPath)
	return nil
}
