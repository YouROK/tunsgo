package p2p

import (
	"log"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

const MaxPeers = 50

type PeerManager struct {
	muPeers sync.RWMutex
	muRelay sync.RWMutex
	peers   map[peer.ID]*PeerInfo
	relays  map[peer.ID]*PeerInfo
}

func NewPeerManager() *PeerManager {
	return &PeerManager{
		peers:  make(map[peer.ID]*PeerInfo),
		relays: make(map[peer.ID]*PeerInfo),
	}
}

func (pm *PeerManager) Update(info *PeerInfo) {
	pm.muPeers.Lock()
	defer pm.muPeers.Unlock()

	if _, exists := pm.peers[info.ID]; exists {
		pm.peers[info.ID] = info
		return
	}

	if len(pm.peers) >= MaxPeers {
		// remove slow
		var target peer.ID
		var maxLatency time.Duration

		for id, p := range pm.peers {
			if p.Latency > maxLatency {
				maxLatency = p.Latency
				target = id
			}
		}

		if target != "" {
			delete(pm.peers, target)
		}
	}

	pm.peers[info.ID] = info
}

func (pm *PeerManager) UpdateRequest(id peer.ID) {
	pm.muPeers.Lock()
	defer pm.muPeers.Unlock()

	if info, exists := pm.peers[id]; exists {
		pm.peers[info.ID].LastReq = time.Now()
	}
}

func (pm *PeerManager) Exist(id peer.ID) bool {
	exists := false
	pm.muPeers.RLock()
	defer pm.muPeers.RUnlock()
	_, exists = pm.peers[id]
	return exists
}

func (pm *PeerManager) GetPeer() peer.ID {
	pm.muPeers.RLock()
	defer pm.muPeers.RUnlock()

	if len(pm.peers) == 0 {
		return ""
	}

	var best peer.ID
	var oldestReq = time.Now()

	for id, info := range pm.peers {
		if info.LastReq.Before(oldestReq) {
			oldestReq = info.LastReq
			best = id
		}
	}
	return best
}

func (pm *PeerManager) Peers() int {
	pm.muPeers.RLock()
	defer pm.muPeers.RUnlock()

	return len(pm.peers)
}

func (pm *PeerManager) Remove(id peer.ID) {
	pm.muPeers.Lock()
	defer pm.muPeers.Unlock()
	delete(pm.peers, id)
}

func (pm *PeerManager) AddRelay(id peer.ID, latency time.Duration) {
	pm.muRelay.Lock()
	defer pm.muRelay.Unlock()

	if _, exists := pm.relays[id]; exists {
		pm.relays[id] = &PeerInfo{
			ID:      id,
			Latency: latency,
		}
		return
	}

	if len(pm.relays) >= 10 {
		var worstPeer peer.ID
		var maxLatency time.Duration = -1

		for p, info := range pm.relays {
			if info.Latency > maxLatency {
				maxLatency = info.Latency
				worstPeer = p
			}
		}

		if latency < maxLatency {
			delete(pm.relays, worstPeer)
			log.Printf("[P2P] Удалено медленное реле %s (ping: %v), добавляем %s (ping: %v)",
				worstPeer, maxLatency, id, latency)
		} else {
			return
		}
	}

	pm.relays[id] = &PeerInfo{
		ID:      id,
		Latency: latency,
	}
}

func (pm *PeerManager) GetRelays() []*PeerInfo {
	pm.muRelay.RLock()
	defer pm.muRelay.RUnlock()
	relays := make([]*PeerInfo, 0, len(pm.relays))
	for _, info := range pm.relays {
		relays = append(relays, info)
	}
	return relays
}
