package net

import (
	"TunsGo/config"
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	"github.com/miekg/dns"
)

type DNS struct {
	tunnels []*Tunnel
	server  *dns.Server
}

func NewDNS(tunnels []*Tunnel) *DNS {
	d := &DNS{tunnels: tunnels}
	return d
}

func (d *DNS) Start() {
	d.initTunnels()
	d.fillRouteTable()

	//serve dns
	mux := dns.NewServeMux()
	mux.HandleFunc(".", func(w dns.ResponseWriter, r *dns.Msg) {
		if len(r.Question) == 0 {
			dns.HandleFailed(w, r)
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

		w.WriteMsg(resp)
	})

	d.server = &dns.Server{
		Addr:    config.Cfg.DNS.Listen,
		Net:     "udp",
		Handler: mux,
	}

	log.Printf("[DNS] Listen %s", config.Cfg.DNS.Listen)
	if err := d.server.ListenAndServe(); err != nil {
		log.Fatalf("[DNS] Error run server: %v", err)
	}
}

func (d *DNS) Stop() {
	for _, t := range d.tunnels {
		t.Cleanup()
	}
	d.server.Shutdown()
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
			log.Println("Error setup tunnel:", t.tun.TunName, err)
			os.Exit(1)
		}
		if !isDnsForward && config.Cfg.DNS.ForwardTun != "" && t.tun.TunName == config.Cfg.DNS.ForwardTun {
			dnsip, _, err := net.SplitHostPort(config.Cfg.DNS.Upstream)
			if err == nil {
				log.Println("Using DNS forward VPN")
				t.AddIP(net.ParseIP(dnsip))
				isDnsForward = true
			} else {
				log.Println("Error parsing DNS forward VPN:", err)
				log.Println("Disabling DNS forward VPN")
			}
		}
	}
}

func (d *DNS) fillRouteTable() {
	if config.Cfg.FillRouteTable {
		for _, t := range d.tunnels {
			log.Println("Filling route table:", t.tun.TunName)
			for _, name := range t.domains {
				rr, err := d.GetIps(name)
				if err == nil {
					for _, r := range rr {
						if rec, ok := r.(*dns.A); ok {
							t.AddIP(rec.A)
						}
					}
				} else {
					log.Println("Error getting domain:", name, err)
				}
			}
		}

		log.Println("Filling route table complete")
	}
}
