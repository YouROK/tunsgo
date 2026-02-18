package cmds

import (
	"encoding/json"
	"log"

	"github.com/libp2p/go-libp2p/core/peer"
)

type handshakeData struct {
	Version       string   `json:"version"`
	ProvidedHosts []string `json:"provided_hosts"`
}

func (s *P2PServer) pingCmd(pid peer.ID) {
	env := &envelope{
		Type:    "ping",
		Payload: nil,
	}

	//start := time.Now()
	resp, err := s.sendToPeer(pid, env)
	if err != nil {
		return
	}
	//latency := time.Since(start)

	if resp.Type == "error" {
		log.Printf("[P2P] Пир %s вернул ошибку: %s", pid, string(resp.Data))
		return
	}

	var info handshakeData
	if err = json.Unmarshal(resp.Data, &info); err != nil {
		log.Printf("[P2P] Ошибка парсинга рукопожатия %s: %v", pid, err)
		return
	}

	//peerInfo := &PeerInfo{
	//	ID:            pid,
	//	Latency:       latency,
	//	Version:       info.Version,
	//	ProvidedHosts: info.ProvidedHosts,
	//	LastReq:       time.Now(),
	//}
	//s.manager.Update(peerInfo)

	//if !s.manager.Exist(pid) {
	//	log.Printf("[P2P] Узел %s пинг: %v", pid.String(), latency)
	//}

	protocols, err := s.host.Peerstore().SupportsProtocols(pid, "/libp2p/circuit/relay/0.2.0/hop")
	if err == nil && len(protocols) > 0 {
		log.Printf("[P2P] Узел %s поддерживает Relay! Сохраняю в кэш...", pid)
		//s.manager.AddRelay(pid, latency)
	}
}
