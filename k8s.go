package main

import (
	"context"
	"fmt"
	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"math/big"
	"net"
	"strings"
	"time"
)

func (plugin *Plugin) initSelf() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	node, err := plugin.client.CoreV1().Nodes().Get(ctx, plugin.nodeName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("cannot get node [%s]: %w", plugin.nodeName, err)
	}
	plugin.nodeIP = ParseNodeIP(node)
	log.Printf("node [%s] has ip [%s]", plugin.nodeName, plugin.nodeIP)
	// Find the pod CIDRs of the node.
	plugin.nodePodCIDRs, err = ParseNodePodCIDRs(node)
	if err != nil {
		return fmt.Errorf("cannot parse pod cidrs of node [%s]: %w", plugin.nodeName, err)
	}
	if len(plugin.nodePodCIDRs) == 0 {
		return fmt.Errorf("node [%s] does not have a pod CIDR", plugin.nodeName)
	}
	// Generate the gateway IPs.
	for _, cidr := range plugin.nodePodCIDRs {
		gatewayIP := net.IP(big.NewInt(0).Add(big.NewInt(0).SetBytes(cidr.IP), big.NewInt(1)).Bytes())
		plugin.gateways = append(plugin.gateways, gatewayIP)
		plugin.gatewayMap[gatewayIP.String()] = true
	}
	log.Printf("node [%s] has gateway ips %v", plugin.nodeName, plugin.gateways)
	return nil
}

func (plugin *Plugin) initCluster() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	kubeadmConfig, err := plugin.client.CoreV1().ConfigMaps("kube-system").Get(ctx, "kubeadm-config", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("cluster pod cidrs not provided, and cannot get kubeadm config")
	}
	clusterConfig := &KubeadmClusterConfiguration{}
	if err := yaml.Unmarshal([]byte(kubeadmConfig.Data["ClusterConfiguration"]), clusterConfig); err != nil {
		return fmt.Errorf("cannot unmarshal kubeadm cluster config: %w", err)
	}
	podSubnet := clusterConfig.Networking.PodSubnet
	if len(podSubnet) == 0 {
		return fmt.Errorf("kubeadm cluster config has empty pod subnet: %w", err)
	}
	podCIDRs, err := ParseIPNets(strings.Split(podSubnet, ","))
	if err != nil {
		return fmt.Errorf("cannot parse pod subnet from kubeadm config: %w", err)
	}
	if len(podCIDRs) == 0 {
		return fmt.Errorf("kubeadm cluster config has empty pod subnet")
	}
	plugin.clusterPodCIDRs = podCIDRs
	return nil
}
