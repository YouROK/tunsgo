package net

import (
	"TunsGo/config"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

type Tunnel struct {
	vpnLink netlink.Link
	ips     sync.Map
	tun     *config.TunConfig
	domains []string
}

func NewTunnel(tun *config.TunConfig) *Tunnel {
	r := &Tunnel{}
	r.tun = tun

	dir := filepath.Dir(os.Args[0])
	fname := filepath.Join(dir, r.tun.TunName+".list")
	buf, err := os.ReadFile(fname)
	if err != nil {
		log.Println("Error reading list domains", fname, err)
		os.Exit(1)
	}
	for _, domain := range strings.Split(string(buf), "\n") {
		domain = strings.TrimSpace(domain)
		if domain != "" {
			r.domains = append(r.domains, strings.ToLower(domain))
		}
	}

	return r
}

func (a *Tunnel) Setup() error {
	link, err := netlink.LinkByName(a.tun.TunName)
	if err != nil {
		return fmt.Errorf("interface %s not found: %v", a.tun.TunName, err)
	}

	a.vpnLink = link

	_, defaultDst, _ := net.ParseCIDR("0.0.0.0/0")
	route := &netlink.Route{
		LinkIndex: link.Attrs().Index,
		Table:     a.tun.TableID,
		Dst:       defaultDst,
		Scope:     netlink.SCOPE_LINK,
	}

	if err := netlink.RouteReplace(route); err != nil {
		return fmt.Errorf("error create table %v: %v", a.tun.TableID, err)
	}

	log.Printf("[ROUTE] Success. Interface: %v, Table: %v\n", a.tun.TunName, a.tun.TableID)
	return nil
}

func (a *Tunnel) AddIP(ip net.IP) {
	ipv4 := ip.To4()
	if ipv4 == nil {
		return
	}

	ipStr := ipv4.String()
	if _, loaded := a.ips.LoadOrStore(ipStr, true); loaded {
		return
	}

	rule := netlink.NewRule()
	rule.Dst = &net.IPNet{IP: ipv4, Mask: net.CIDRMask(32, 32)}
	rule.Table = a.tun.TableID
	rule.Priority = 100

	if err := netlink.RuleAdd(rule); err != nil {
		log.Printf("[ERROR] Error RuleAdd for %s: %v", ipStr, err)
		a.ips.Delete(ipStr)
		return
	}

	log.Printf("[ROUTE] IP %s forward in VPN", ipStr)
}

func (a *Tunnel) Cleanup() {
	if a == nil {
		return
	}
	log.Println("[ROUTE] Remove rules...")

	rules, err := netlink.RuleList(unix.AF_INET)
	if err != nil {
		log.Printf("[ERROR] Failed to get the list of rules: %v", err)
		return
	}

	for _, r := range rules {
		if r.Table == a.tun.TableID {
			_ = netlink.RuleDel(&r)
		}
	}

	_, defaultDst, _ := net.ParseCIDR("0.0.0.0/0")
	route := &netlink.Route{
		Table: a.tun.TableID,
		Dst:   defaultDst,
	}
	_ = netlink.RouteDel(route)

	log.Println("[ROUTE] Cleanup end")
}
