package urlproxy

import (
	"encoding/json"
	"log"
	"slices"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/yourok/tunsgo/p2p/models"
)

func (p *UrlProxy) announceHosts() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		ann := models.PeerInfo{
			PeerID:    p.host.ID().String(),
			Hosts:     p.opts.Hosts,
			Timestamp: time.Now().Unix(),
		}

		data, _ := json.Marshal(ann)
		err := p.topic.Publish(p.ctx, data)
		if err != nil {
			log.Println("[PUB] Error announce to room:", err)
		}

		select {
		case <-ticker.C:
		case <-p.ctx.Done():
			return
		}
	}
}

func (p *UrlProxy) listenForAnnouncements() {
	sub, _ := p.topic.Subscribe()
	log.Println("[PUB] Listen announcements...")
	for {
		select {
		case <-p.ctx.Done():
			return
		default:
			msg, err := sub.Next(p.ctx)
			if err != nil {
				return
			}
			if msg.ReceivedFrom == p.host.ID() {
				continue
			}

			var info *models.PeerInfo
			if err := json.Unmarshal(msg.Data, &info); err == nil {
				p.addPeer(info)
			}
		}
	}
}

func (p *UrlProxy) addPeer(info *models.PeerInfo) {
	pID, _ := peer.Decode(info.PeerID)
	info.LastSeen = time.Now()
	p.muPeers.Lock()
	if infoS, ok := p.peers[pID]; !ok || !slices.Equal(infoS.Hosts, info.Hosts) {
		log.Printf("[PUB] Node %s have hosts %v", info.PeerID, info.Hosts)
	}
	p.peers[pID] = info
	p.muPeers.Unlock()
	p.host.ConnManager().UpsertTag(pID, "tuns-node", func(current int) int {
		return 60
	})
}

func (p *UrlProxy) cleanupPeersGC() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.cleanupPeers()
		case <-p.ctx.Done():
			return
		}
	}
}

func (p *UrlProxy) cleanupPeers() {
	p.muPeers.Lock()
	defer p.muPeers.Unlock()

	now := time.Now()
	connectedPeers := p.host.Network().Peers()

	connectedMap := make(map[peer.ID]bool)
	for _, pID := range connectedPeers {
		connectedMap[pID] = true
	}

	for pID, info := range p.peers {
		isConnected := connectedMap[pID]
		if !isConnected && now.Sub(info.LastSeen) > 5*time.Minute {
			log.Printf("[P2P] GC: removed unusable node from list %p", pID)
			delete(p.peers, pID)
		}
	}
}
