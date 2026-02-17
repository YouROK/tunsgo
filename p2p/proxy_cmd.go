package p2p

import (
	"encoding/json"
	"fmt"
	"log"
)

type proxyCommand struct {
	URL string
}

func (s *P2PServer) SendRequestProxy(targetURL string) ([]byte, error) {
	proxyCmd := proxyCommand{
		URL: targetURL,
	}
	cmdPayload, err := json.Marshal(proxyCmd)
	if err != nil {
		return nil, fmt.Errorf("ошибка маршалинга команды: %w", err)
	}

	env := &envelope{
		Type:    "proxy",
		Payload: cmdPayload,
	}

	for {
		bestPeer := s.manager.GetPeer()
		if bestPeer == "" {
			return nil, fmt.Errorf("нет доступных пиров для проксирования")
		}

		log.Println("[P2P Proxy] Отсылаю запрос к пиру", bestPeer.String())

		s.manager.UpdateRequest(bestPeer)

		resp, err := s.sendToPeer(bestPeer, env)
		if err != nil {
			continue
		}

		if resp.Type == "error" {
			return nil, fmt.Errorf("удаленный пир вернул ошибку: %s", string(resp.Data))
		}

		if resp.Type == "busy" {
			log.Println("[P2P Proxy] Узел занят ищем другой", bestPeer.String())
			continue
		}

		return resp.Data, nil
	}
}
