package p2p

import (
	"encoding/json"
	"log"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

func (s *P2PServer) announceHosts() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		ann := peerInfo{
			PeerID:    s.host.ID().String(),
			Hosts:     s.cfg.Hosts.ProvidedHosts,
			Timestamp: time.Now().Unix(),
		}

		log.Println("[PubSub] Анонс в комнату")
		data, _ := json.Marshal(ann)
		err := s.topic.Publish(s.ctx, data)
		if err != nil {
			log.Println("[PubSub] Ошибка анонса в комнату:", err)
		}

		select {
		case <-ticker.C:
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *P2PServer) listenForAnnouncements() {
	sub, _ := s.topic.Subscribe()
	log.Println("[PubSub] Слушаем комнату...")
	go s.cleanupPeersGC()
	for {
		msg, err := sub.Next(s.ctx)
		if err != nil {
			return
		}
		if msg.ReceivedFrom == s.host.ID() {
			continue
		}

		var info *peerInfo
		if err := json.Unmarshal(msg.Data, &info); err == nil {
			s.addPeer(info)
			log.Printf("[P2P] Узел %s имеет хосты %v", info.PeerID, info.Hosts)
		}
	}
}

func (s *P2PServer) addPeer(info *peerInfo) {
	pID, _ := peer.Decode(info.PeerID)
	info.LastSeen = time.Now()
	s.muPeers.Lock()
	s.peers[pID] = info
	s.muPeers.Unlock()
	s.host.ConnManager().UpsertTag(pID, "tuns-node", func(current int) int {
		return 40
	})
}

func (s *P2PServer) cleanupPeersGC() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.cleanupPeers()
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *P2PServer) cleanupPeers() {
	s.muPeers.Lock()
	defer s.muPeers.Unlock()

	now := time.Now()
	connectedPeers := s.host.Network().Peers()

	connectedMap := make(map[peer.ID]bool)
	for _, pID := range connectedPeers {
		connectedMap[pID] = true
	}

	for pID, info := range s.peers {
		isConnected := connectedMap[pID]
		if !isConnected && now.Sub(info.LastSeen) > 5*time.Minute {
			log.Printf("[P2P] GC: удаление неактивного пира %s", pID)
			delete(s.peers, pID)
		}
	}
}
