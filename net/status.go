package net

import (
	"fmt"
	"net"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/mackerelio/go-osstat/cpu"
	"github.com/mackerelio/go-osstat/memory"
	"github.com/mackerelio/go-osstat/network"
	"github.com/mackerelio/go-osstat/uptime"
)

type InterfaceStatus struct {
	Name      string   `json:"name"`
	Addresses []string `json:"addresses"`
	MTU       int      `json:"mtu"`
	IsUp      bool     `json:"is_up"`
	IsManaged bool     `json:"is_managed"`

	// Дополнительные поля для роутеров:
	MAC        string `json:"mac"`      // Помогает идентифицировать физ. порт
	RxBytes    string `json:"rx_bytes"` // Принято байт (счётчик)
	TxBytes    string `json:"tx_bytes"` // Отправлено байт
	IsLoopback bool   `json:"isloopback"`

	// Данные из конфига TunsGo
	TableID   int      `json:"table_id,omitempty"`
	FWMark    int      `json:"fw_mark,omitempty"`
	Domains   []string `json:"domains,omitempty"`
	TargetIPs []string `json:"target_ips,omitempty"`
}

type OSStatus struct {
	Uptime string `json:"uptime"`
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
}

type Status struct {
	Interfaces []*InterfaceStatus `json:"interfaces"`
	OSStatus   *OSStatus          `json:"os_status"`
}

func (m *Manager) GetSystemStatus() (Status, error) {
	var status Status
	ifaces, err := net.Interfaces()
	if err != nil {
		return status, err
	}

	stats, _ := network.Get()

	status.OSStatus = getOSStatus()

	for _, iface := range ifaces {
		iStatus := &InterfaceStatus{
			Name: iface.Name,
			MTU:  iface.MTU,
			MAC:  iface.HardwareAddr.String(),
			IsUp: (iface.Flags & net.FlagUp) != 0,
		}

		if iface.Flags&net.FlagLoopback != 0 {
			iStatus.IsLoopback = true
		} else {
			iStatus.IsLoopback = false
		}

		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			iStatus.Addresses = append(iStatus.Addresses, addr.String())
		}

		for _, s := range stats {
			if s.Name != iface.Name {
				iStatus.RxBytes = humanize.IBytes(s.RxBytes)
				iStatus.TxBytes = humanize.IBytes(s.TxBytes)
			}
		}

		for _, t := range m.dns.tunnels {
			if t.tun.TunName == iface.Name {
				iStatus.IsManaged = true
				iStatus.TableID = t.tun.TableID
				iStatus.FWMark = int(t.tun.ForwardMark)
				iStatus.Domains = t.domains
				t.ips.Range(func(key, value interface{}) bool {
					if ip, ok := key.(string); ok {
						iStatus.TargetIPs = append(iStatus.TargetIPs, ip)
					}
					return true
				})
			}
		}
		status.Interfaces = append(status.Interfaces, iStatus)
	}
	return status, nil
}

func getOSStatus() *OSStatus {
	stat := &OSStatus{}

	mem, err := memory.Get()
	if err == nil {
		stat.Memory = humanize.IBytes(mem.Free) + " / " + humanize.IBytes(mem.Total)
	}

	up, err := uptime.Get()
	if err == nil {
		stat.Uptime = up.String()
	}

	before, _ := cpu.Get()
	time.Sleep(time.Second)
	after, _ := cpu.Get()

	total := float64(after.Total - before.Total)
	busy := (total - float64(after.Idle-before.Idle)) / total * 100
	stat.CPU = fmt.Sprintf("%.2f%%", busy)

	return stat
}
