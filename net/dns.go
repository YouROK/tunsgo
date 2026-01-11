package net

import (
	"TunsGo/config"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
)

type cacheItem struct {
	msg        *dns.Msg
	expiration time.Time
}

type DNS struct {
	tunnels []*Tunnel
	server  *dns.Server
	cache   map[string]cacheItem
	mu      sync.RWMutex
}

func NewDNS(tunnels []*Tunnel) *DNS {
	return &DNS{
		tunnels: tunnels,
		cache:   make(map[string]cacheItem),
	}
}

func (d *DNS) Start() {
	d.initTunnels()
	d.fillRouteTable()

	go d.cacheCleaner()

	mux := dns.NewServeMux()
	mux.HandleFunc(".", d.handleDNSRequest)

	d.server = &dns.Server{
		Addr:    config.Cfg.DNS.Listen,
		Net:     "udp",
		Handler: mux,
	}

	if config.Cfg.DNS.UseCache {
		log.Println("[DNS] Using cache enabled")
	}

	log.Printf("[DNS] Listen %s", config.Cfg.DNS.Listen)
	if err := d.server.ListenAndServe(); err != nil {
		log.Fatalf("[DNS] Error run server: %v", err)
	}
}

func (d *DNS) handleDNSRequest(w dns.ResponseWriter, r *dns.Msg) {
	if len(r.Question) == 0 {
		dns.HandleFailed(w, r)
		return
	}

	if cachedResp := d.getFromCache(r); cachedResp != nil {
		w.WriteMsg(cachedResp)
		return
	}

	q := r.Question[0]
	name := strings.ToLower(q.Name)

	resp, err := dns.Exchange(r, config.Cfg.DNS.Upstream)
	if err != nil {
		log.Printf("[DNS] Error request %s: %v", name, err)
		dns.HandleFailed(w, r)
		return
	}

	router := d.findRouter(name)
	if router != nil && q.Qtype == dns.TypeA {
		for _, ans := range resp.Answer {
			if rec, ok := ans.(*dns.A); ok {
				router.AddIP(rec.A)
			}
		}
	}

	d.setInCache(r, resp)

	w.WriteMsg(resp)
}

func (d *DNS) getFromCache(r *dns.Msg) *dns.Msg {
	q := r.Question[0]
	key := fmt.Sprintf("%s-%d", strings.ToLower(q.Name), q.Qtype)

	d.mu.RLock()
	item, ok := d.cache[key]
	d.mu.RUnlock()

	if ok {
		if time.Now().Before(item.expiration) {
			resp := item.msg.Copy()
			resp.Id = r.Id
			return resp
		}
	}
	return nil
}

func (d *DNS) setInCache(r *dns.Msg, resp *dns.Msg) {
	if len(resp.Answer) == 0 {
		return
	}

	q := r.Question[0]
	key := fmt.Sprintf("%s-%d", strings.ToLower(q.Name), q.Qtype)

	var minTTL uint32 = 0
	for i, ans := range resp.Answer {
		ttl := ans.Header().Ttl
		if i == 0 || ttl < minTTL {
			minTTL = ttl
		}
	}

	if minTTL < 10 {
		minTTL = 10
	}

	d.mu.Lock()
	d.cache[key] = cacheItem{
		msg:        resp.Copy(),
		expiration: time.Now().Add(time.Duration(minTTL) * time.Second),
	}
	d.mu.Unlock()
}

func (d *DNS) cacheCleaner() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		d.mu.Lock()
		now := time.Now()
		before := len(d.cache)
		for key, item := range d.cache {
			if now.After(item.expiration) {
				delete(d.cache, key)
			}
		}
		after := len(d.cache)
		d.mu.Unlock()

		if before != after {
			log.Printf("[DNS] Cleaned %d expired items", before-after)
		}
	}
}

func (d *DNS) Stop() {
	for _, t := range d.tunnels {
		t.Cleanup()
	}
	if d.server != nil {
		d.server.Shutdown()
	}
}

func (d *DNS) GetIps(domain string) ([]dns.RR, error) {
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(domain), dns.TypeA)
	msg.RecursionDesired = true

	client := new(dns.Client)
	response, _, err := client.Exchange(msg, config.Cfg.DNS.Upstream)
	if err != nil {
		return nil, err
	}

	if response.Rcode != dns.RcodeSuccess {
		return nil, fmt.Errorf("error request dns %v", response.Rcode)
	}

	return response.Answer, nil
}

func (d *DNS) findRouter(domain string) *Tunnel {
	domain = strings.ToLower(strings.TrimSuffix(domain, "."))
	for _, router := range d.tunnels {
		for _, name := range router.domains {
			if strings.HasSuffix(domain, name) {
				return router
			}
		}
	}
	return nil
}

func (d *DNS) initTunnels() {
	isDnsForward := false
	for _, t := range d.tunnels {
		err := t.Setup()
		if err != nil {
			log.Println("[DNS] Error setup tunnel:", t.tun.TunName, err)
			os.Exit(1)
		}
		if !isDnsForward && config.Cfg.DNS.ForwardTun != "" && t.tun.TunName == config.Cfg.DNS.ForwardTun {
			dnsip, _, err := net.SplitHostPort(config.Cfg.DNS.Upstream)
			if err == nil {
				log.Println("[DNS] Using DNS forward VPN")
				t.AddIP(net.ParseIP(dnsip))
				isDnsForward = true
			}
		}
	}
}

func (d *DNS) fillRouteTable() {
	if config.Cfg.FillRouteTable {
		for _, t := range d.tunnels {
			log.Println("[DNS] Filling route table:", t.tun.TunName)
			for _, name := range t.domains {
				rr, err := d.GetIps(name)
				if err == nil {
					for _, r := range rr {
						if rec, ok := r.(*dns.A); ok {
							t.AddIP(rec.A)
						}
					}
				}
			}
		}
		log.Println("[DNS] Filling route table complete")
	}
}

func (d *DNS) GetCacheSize() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.cache)
}
