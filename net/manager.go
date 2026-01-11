package net

import (
	"TunsGo/config"
)

type Manager struct {
	dns *DNS
}

func NewManager() *Manager {
	var tunnels []*Tunnel
	for _, tun := range config.Cfg.Tuns {
		if tun.Disable {
			continue
		}
		t := NewTunnel(tun)
		tunnels = append(tunnels, t)
	}
	d := NewDNS(tunnels)
	m := &Manager{dns: d}
	return m
}

func (m *Manager) Start() {
	go m.dns.Start()
}

func (m *Manager) Stop() {
	m.dns.Stop()
}
