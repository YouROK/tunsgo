package pex

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/yourok/tunsgo/opts"
	"github.com/yourok/tunsgo/p2p/models"
)

type Pex struct {
	host       host.Host
	opts       *opts.Options
	ctx        context.Context
	lastSeeded map[peer.ID]time.Time
}

func NewPex(c *models.SrvCtx) *Pex {
	return &Pex{
		host:       c.Host,
		opts:       c.Opts,
		ctx:        c.Ctx,
		lastSeeded: make(map[peer.ID]time.Time),
	}
}

func (p *Pex) Start() error {
	log.Println("[PEX] Service started")
	go p.discoveryLoop()
	return nil
}

func (p *Pex) Stop() {
	log.Println("[PEX] Service stopped")
}

func (p *Pex) Name() string {
	return "PEX"
}

func (p *Pex) ProtocolID() protocol.ID {
	return "/tunsgo/pex/1.0.0"
}

func (p *Pex) HandleStream(stream network.Stream) {
	defer stream.Close()

	remotePeer := stream.Conn().RemotePeer()
	tunsGoPeers := make([]peer.AddrInfo, 0, 30)

	added := make(map[peer.ID]bool)
	added[p.host.ID()] = true
	added[remotePeer] = true

	for _, conn := range p.host.Network().Conns() {
		pid := conn.RemotePeer()

		if !added[pid] && p.isTunsGoPeer(pid) {
			info := p.host.Peerstore().PeerInfo(pid)
			if len(info.Addrs) > 0 {
				tunsGoPeers = append(tunsGoPeers, info)
				added[pid] = true
			}
		}

		if len(tunsGoPeers) >= 30 {
			break
		}
	}

	if len(tunsGoPeers) < 30 {
		allKnown := p.host.Peerstore().Peers()
		for _, pid := range allKnown {
			if !added[pid] && p.isTunsGoPeer(pid) {
				info := p.host.Peerstore().PeerInfo(pid)
				if len(info.Addrs) > 0 {
					tunsGoPeers = append(tunsGoPeers, info)
					added[pid] = true
				}
			}

			if len(tunsGoPeers) >= 30 {
				break
			}
		}
	}

	json.NewEncoder(stream).Encode(tunsGoPeers)
}

func (p *Pex) discoveryLoop() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			currentConns := p.host.Network().Peers()
			tunsCount := 0
			for _, pid := range currentConns {
				if p.isTunsGoPeer(pid) {
					tunsCount++
				}
			}

			if tunsCount < p.opts.P2P.LowConns {
				p.askNeighbors()
			}
		}
	}
}

func (p *Pex) askNeighbors() {
	conns := p.host.Network().Conns()
	for _, conn := range conns {
		remoteID := conn.RemotePeer()

		if last, ok := p.lastSeeded[remoteID]; ok && time.Since(last) < 5*time.Minute {
			continue
		}

		protocols, err := p.host.Peerstore().SupportsProtocols(remoteID, p.ProtocolID())
		if err == nil && len(protocols) > 0 {
			go p.requestPeersFrom(remoteID)
		}
	}
}

func (p *Pex) requestPeersFrom(id peer.ID) {
	ctx, cancel := context.WithTimeout(p.ctx, 10*time.Second)
	defer cancel()

	stream, err := p.host.NewStream(ctx, id, p.ProtocolID())
	if err != nil {
		return
	}
	defer stream.Close()

	var discovered []peer.AddrInfo
	if err := json.NewDecoder(stream).Decode(&discovered); err != nil {
		return
	}

	p.lastSeeded[id] = time.Now()

	for _, info := range discovered {
		if info.ID == p.host.ID() {
			continue
		}

		p.host.Peerstore().AddAddrs(info.ID, info.Addrs, time.Hour)
		log.Printf("[PEX] Found tunsgo node: %s from %s", info.ID, id.ShortString())
	}
}

func (p *Pex) isTunsGoPeer(id peer.ID) bool {
	protocols, err := p.host.Peerstore().GetProtocols(id)
	if err != nil {
		return false
	}
	for _, proto := range protocols {
		if strings.HasPrefix(string(proto), "/tunsgo") {
			return true
		}
	}
	return false
}
