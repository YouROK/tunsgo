package p2p

import (
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

const MaxPeers = 50

type PeerManager struct {
	mu    sync.RWMutex
	peers map[peer.ID]*PeerInfo
}

func NewPeerManager() *PeerManager {
	return &PeerManager{peers: make(map[peer.ID]*PeerInfo)}
}

func (pm *PeerManager) Update(info *PeerInfo) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

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
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if info, exists := pm.peers[id]; exists {
		pm.peers[info.ID].LastReq = time.Now()
	}
}

func (pm *PeerManager) Exist(id peer.ID) bool {
	exists := false
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	_, exists = pm.peers[id]
	return exists
}

func (pm *PeerManager) GetPeer() peer.ID {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

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

func (pm *PeerManager) Remove(id peer.ID) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	delete(pm.peers, id)
}
