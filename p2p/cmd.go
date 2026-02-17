package p2p

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

type envelope struct {
	Type    string `json:"type"` // "proxy", "ping", "status"
	Payload []byte `json:"payload,omitempty"`
}

type response struct {
	Type string `json:"type"` // "proxy", "ping", "status", "error"
	Data []byte `json:"data,omitempty"`
}

func (s *P2PServer) sendToPeer(pid peer.ID, req *envelope) (*response, error) {
	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()

	stream, err := s.host.NewStream(ctx, pid, ProtocolID)
	if err != nil {
		log.Printf("[P2P] Ошибка соединения с узлом %s: %v", pid, err)
		s.manager.Remove(pid)
		return nil, err
	}
	defer stream.Close()

	envBytes, _ := json.Marshal(req)

	if _, err = stream.Write(envBytes); err != nil {
		log.Printf("[P2P] Ошибка отправки сообщения %s: %v", pid, err)
		s.manager.Remove(pid)
		return nil, err
	}
	stream.CloseWrite()

	var resp response
	if err = json.NewDecoder(stream).Decode(&resp); err != nil {
		log.Printf("[P2P] Ошибка декодирования ответа от %s: %v", pid, err)
		return nil, err
	}

	return &resp, nil
}
