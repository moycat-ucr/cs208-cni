package main

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"log"
	"time"
)

func (plugin *Plugin) Listen() error {
watchLoop:
	for {
		watcher, err := plugin.client.CoreV1().Nodes().Watch(context.Background(), metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("cannot watch node list: %w", err)
		}
		watcherCh := watcher.ResultChan()
		log.Print("start listening events")
		for {
			select {
			case event, ok := <-watcherCh:
				if !ok {
					// Watch channel closed. Retry.
					continue watchLoop
				}
				plugin.handle(event)
			}
		}
	}
}

func (plugin *Plugin) handle(event watch.Event) {
	node, ok := event.Object.(*corev1.Node)
	if !ok || node.Name == plugin.nodeName {
		return
	}
	switch event.Type {
	case watch.Added:
		if plugin.addNode(node) {
			plugin.update()
		}
	case watch.Modified:
		if plugin.updateNode(node) {
			plugin.update()
		}
	case watch.Deleted:
		if plugin.deleteNode(node) {
			plugin.update()
		}
	}
}

func (plugin *Plugin) update() {
	nodes := plugin.cloneNodes()
	plugin.applyTunnels(nodes)
	plugin.applyRoutes(nodes)
}

func (plugin *Plugin) addNode(node *corev1.Node) bool {
	log.Printf("adding a node [%s]", node.Name)
	nodes := plugin.cloneNodes()
	if _, ok := nodes[node.Name]; ok {
		return plugin.updateNode(node)
	}
	nodeIP := ParseNodeIP(node)
	if nodeIP == nil {
		log.Printf("node [%s] has no ip address", node.Name)
		return false
	}
	nodePodCIDRs, err := ParseNodePodCIDRs(node)
	if err != nil {
		log.Printf("cannot parse the pod cidrs of node [%s]: %v", node.Name, err)
		return false
	}
	parsedNode := &Node{
		Name:     node.Name,
		IP:       nodeIP,
		PodCIDRs: nodePodCIDRs,
		Tunnel:   tunnelPrefix + fmt.Sprintf("%d", time.Now().UnixNano()%100000),
	}
	nodes[node.Name] = parsedNode
	plugin.saveNodes(nodes)
	log.Printf("added node [%s] and dumped map", node.Name)
	return true
}

func (plugin *Plugin) updateNode(node *corev1.Node) bool {
	log.Printf("modifying a node [%s]", node.Name)
	nodes := plugin.cloneNodes()
	oldNode, ok := nodes[node.Name]
	if !ok {
		log.Printf("node [%s] does not exist, adding", node.Name)
		return plugin.addNode(node)
	}
	nodeIP := ParseNodeIP(node)
	nodePodCIDRs, err := ParseNodePodCIDRs(node)
	if err != nil {
		log.Printf("cannot parse pod cidrs of node [%s]: %v", node.Name, err)
		return false
	}
	parsedNode := &Node{
		Name:     node.Name,
		IP:       nodeIP,
		PodCIDRs: nodePodCIDRs,
		Tunnel:   tunnelPrefix + fmt.Sprintf("%d", time.Now().UnixNano()%100000),
	}
	if !parsedNode.HasUpdates(oldNode) {
		return false
	}
	nodes[node.Name] = parsedNode
	plugin.saveNodes(nodes)
	log.Printf("updated node [%s] and saved map", node.Name)
	return true
}

func (plugin *Plugin) deleteNode(node *corev1.Node) bool {
	log.Printf("deleting a node [%s]", node.Name)
	nodes := plugin.cloneNodes()
	if _, ok := nodes[node.Name]; !ok {
		log.Printf("deleting node [%s] which is not present", node.Name)
		return false
	}
	delete(nodes, node.Name)
	plugin.saveNodes(nodes)
	log.Printf("deleted node [%s] and dumped map", node.Name)
	return true
}
