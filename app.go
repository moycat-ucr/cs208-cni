package main

import (
	"fmt"
	"k8s.io/client-go/kubernetes"
	"log"
	"net"
	"sync"
)

type Plugin struct {
	client          kubernetes.Interface
	nodeName        string
	nodeIP          net.IP
	nodePodCIDRs    []*net.IPNet
	clusterPodCIDRs []*net.IPNet
	gateways        []net.IP
	gatewayMap      map[string]bool
	nodes           Nodes
	nodeLock        sync.Mutex
}

func NewPlugin(client kubernetes.Interface, nodeName string) (*Plugin, error) {
	app := &Plugin{
		client:     client,
		nodeName:   nodeName,
		nodes:      make(Nodes),
		gatewayMap: make(map[string]bool),
	}
	if err := app.initSelf(); err != nil {
		return nil, fmt.Errorf("cannot get info about self: %w", err)
	}
	if err := app.initCluster(); err != nil {
		return nil, fmt.Errorf("cannot get info about the cluster: %w", err)
	}
	if err := initCNI(app.nodePodCIDRs); err != nil {
		return nil, fmt.Errorf("cannot init cni: %w", err)
	}
	if err := initNAT(app.clusterPodCIDRs); err != nil {
		return nil, fmt.Errorf("cannot init nat: %w", err)
	}
	log.Print("app initialized")
	return app, nil
}

func (plugin *Plugin) cloneNodes() Nodes {
	plugin.nodeLock.Lock()
	nodes := plugin.nodes
	plugin.nodeLock.Unlock()
	newNodes := make(Nodes, len(nodes))
	for k, v := range nodes {
		node := *v
		newNodes[k] = &node
	}
	return newNodes
}

func (plugin *Plugin) saveNodes(nodes Nodes) {
	plugin.nodeLock.Lock()
	plugin.nodes = nodes
	plugin.nodeLock.Unlock()
}
