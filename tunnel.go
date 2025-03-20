package main

import (
	"github.com/vishvananda/netlink"
	"log"
	"net"
	"reflect"
	"strings"
	"syscall"
)

func (plugin *Plugin) applyTunnels(nodes Nodes) {
	log.Print("applying tunnels")
	linkMap := make(map[string]*netlink.Iptun)
	tunnelMap := make(Nodes, len(nodes))
	for _, node := range nodes {
		tunnelMap[node.Tunnel] = node
	}
	links, err := netlink.LinkList()
	if err != nil {
		log.Printf("cannot list links: %v", err)
	}
	for _, link := range links {
		link, ok := link.(*netlink.Iptun)
		if !ok {
			continue
		}
		linkName := link.Attrs().Name
		if strings.HasPrefix(linkName, tunnelPrefix) {
			if _, ok := tunnelMap[linkName]; ok {
				linkMap[linkName] = link
			} else {
				log.Printf("removing unknown tunnel %s", linkName)
				if err := netlink.LinkDel(link); err != nil {
					log.Printf("cannot delete tunnel: %v", err)
				}
			}
		}
	}
	for linkName, node := range tunnelMap {
		link, ok := linkMap[linkName]
		if ok {
			if plugin.isTunnelGood(link, node) {
				log.Printf("tunnel [%s] to node [%s] is good, skipping", linkName, node.Name)
				continue
			}
			log.Printf("tunnel [%s] to node [%s] is not good, recreating", linkName, node.Name)
			if err := netlink.LinkDel(link); err != nil {
				log.Printf("cannot delete stale tunnel [%s] to node [%s]: %v", linkName, node.Name, err)
				continue
			}
		}
		log.Printf("creating tunnel [%s] to node [%s] (%v)", linkName, node.Name, node.IP)
		link = &netlink.Iptun{
			LinkAttrs: netlink.LinkAttrs{Name: linkName},
			Local:     plugin.nodeIP,
			Remote:    node.IP,
		}
		if err := netlink.LinkAdd(link); err != nil {
			log.Printf("cannot create tunnel [%s]: %v", linkName, err)
			continue
		}
		for _, gatewayIP := range plugin.gateways {
			if err := netlink.AddrAdd(link, &netlink.Addr{
				IPNet: &net.IPNet{
					IP:   gatewayIP,
					Mask: net.CIDRMask(len(gatewayIP)<<3, len(gatewayIP)<<3),
				},
			}); err != nil {
				log.Printf("cannot add address [%s] to tunnel [%s]: %v", gatewayIP, linkName, err)
				continue
			}
		}
		if err := netlink.LinkSetUp(link); err != nil {
			log.Printf("cannot bring tunnel [%s] up: %v", linkName, err)
			continue
		}
	}
}

func (plugin *Plugin) applyRoutes(nodes Nodes) {
	log.Print("applying routes")
	for _, node := range nodes {
		link, err := netlink.LinkByName(node.Tunnel)
		if err != nil {
			log.Printf("cannot get tunnel [%s] to node [%s]: %v", node.Tunnel, node.Name, err)
			continue
		}
		log.Printf("checking routes of tunnel [%s] to node [%s]", node.Tunnel, node.Name)
		routeMap := make(map[string]*net.IPNet)
		for _, ipNet := range node.PodCIDRs {
			routeMap[ipNet.String()] = ipNet
		}
		routes, err := netlink.RouteList(link, netlink.FAMILY_ALL)
		if err != nil {
			log.Printf("cannot list routes of tunnel [%s] to node [%s]: %v", node.Tunnel, node.Name, err)
			continue
		}
		for _, route := range routes {
			if route.Dst != nil && route.Src == nil && len(route.Gw) == 0 && routeMap[route.Dst.String()] != nil {
				log.Printf("route to [%s] on node [%s] via tunnel [%s] exists",
					route.Dst.String(), node.Name, node.Tunnel)
				delete(routeMap, route.Dst.String())
				continue
			}
			log.Printf("deleting unexpected route on tunnel [%s]: %v", node.Tunnel, route)
			if err := netlink.RouteDel(&route); err != nil {
				log.Printf("cannot delete route on tunnel [%s]: %v", node.Tunnel, err)
				continue
			}
		}
		for _, routeToAdd := range routeMap {
			log.Printf("adding route to [%s] on node [%s] via tunnel [%s]",
				routeToAdd.String(), node.Name, node.Tunnel)
			route := netlink.Route{
				LinkIndex: link.Attrs().Index,
				Dst:       routeToAdd,
			}
			if err := netlink.RouteAdd(&route); err != nil {
				log.Printf("cannot add route to [%s] on node [%s] via tunnel [%s]: %v",
					routeToAdd.String(), node.Name, node.Tunnel, err)
				continue
			}
		}
	}
}

func (plugin *Plugin) isTunnelGood(link *netlink.Iptun, node *Node) bool {
	if link.LinkAttrs.Flags|net.FlagUp == 0 {
		return false
	}
	if !link.Local.Equal(plugin.nodeIP) || !link.Remote.Equal(node.IP) {
		return false
	}
	addrs, err := netlink.AddrList(link, netlink.FAMILY_ALL)
	if err != nil {
		return false
	}
	addrMap := make(map[string]bool)
	for _, addr := range addrs {
		if addr.Scope != syscall.RT_SCOPE_UNIVERSE {
			continue
		}
		addrMap[addr.IP.String()] = true
	}
	if !reflect.DeepEqual(addrMap, plugin.gatewayMap) {
		return false
	}
	return true
}
