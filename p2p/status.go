package p2p

import (
	"strings"
	"time"
)

type PeerDetail struct {
	ID        string    `json:"id"`
	Addrs     []string  `json:"addrs"`
	IsTuns    bool      `json:"is_tuns"`             // Наш ли это узел
	Latency   string    `json:"latency"`             // Пинг
	LastSeen  time.Time `json:"last_seen,omitempty"` // Из вашей таблицы peers
	Protocols []string  `json:"protocols"`           // Какие протоколы поддерживает
}

type P2PStatus struct {
	PeerID         string              `json:"peer_id"`
	ListenAddrs    []string            `json:"listen_addrs"`
	TotalConns     int                 `json:"total_conns"`      // Все соединения в сети
	TunsPeersCount int                 `json:"tuns_peers_count"` // Только наши прокси-узлы
	KnownDomains   map[string][]string `json:"known_domains"`    // Статистика: какой домен кто держит
	Peers          []PeerDetail        `json:"peers_list"`
	OurPeers       []PeerDetail        `json:"our_peers_list"`
}

func (s *P2PServer) Status() *P2PStatus {
	s.muPeers.RLock()
	defer s.muPeers.RUnlock()

	status := &P2PStatus{
		PeerID:       s.host.ID().String(),
		TotalConns:   len(s.host.Network().Peers()),
		KnownDomains: make(map[string][]string),
		Peers:        []PeerDetail{},
	}

	// 1. Наши адреса, по которым нас можно найти
	for _, addr := range s.host.Addrs() {
		status.ListenAddrs = append(status.ListenAddrs, addr.String())
	}

	// 2. Обходим все активные соединения в сети libp2p
	connectedPeers := s.host.Network().Peers()
	for _, pID := range connectedPeers {
		// Узнаем, какие протоколы поддерживает узел
		protocols, _ := s.host.Peerstore().GetProtocols(pID)

		var protocolsString []string
		// Проверяем, есть ли наш протокол в списке
		isOurNode := false
		for _, proto := range protocols {
			protocolsString = append(protocolsString, string(proto))
			if strings.HasPrefix(string(proto), "/tunsgo/") {
				isOurNode = true
			}
		}

		if isOurNode {
			status.TunsPeersCount++
		}

		// Собираем инфо о пире
		detail := PeerDetail{
			ID:        pID.String(),
			IsTuns:    isOurNode,
			Protocols: protocolsString,
		}

		// Добавляем адреса из Peerstore
		for _, a := range s.host.Peerstore().Addrs(pID) {
			detail.Addrs = append(detail.Addrs, a.String())
		}

		// Если пир есть в нашей таблице анонсов, берем LastSeen
		if info, ok := s.peers[pID]; ok {
			detail.LastSeen = info.LastSeen
			// Заодно наполняем карту доменов для статистики
			for _, domain := range info.Hosts {
				status.KnownDomains[domain] = append(status.KnownDomains[domain], pID.ShortString())
			}
		}

		status.Peers = append(status.Peers, detail)
		if isOurNode {
			status.OurPeers = append(status.OurPeers, detail)
		}
	}

	return status
}
