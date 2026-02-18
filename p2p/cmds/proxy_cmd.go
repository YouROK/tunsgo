package cmds

import (
	"fmt"
)

type proxyCommand struct {
	URL string
}

func (s *P2PServer) SendRequestProxy(targetURL string) ([]byte, error) {
	//proxyCmd := proxyCommand{
	//	URL: targetURL,
	//}
	//cmdPayload, err := json.Marshal(proxyCmd)
	//if err != nil {
	//	return nil, fmt.Errorf("ошибка маршалинга команды: %w", err)
	//}

	//env := &envelope{
	//	Type:    "proxy",
	//	Payload: cmdPayload,
	//}

	//count := s.manager.PeersCount()

	//for i := 0; i < count; i++ {
	//	bestPeer := s.manager.GetLastPeer()
	//	if bestPeer == nil {
	//		return nil, fmt.Errorf("нет доступных пиров для проксирования")
	//	}
	//
	//	log.Println("[P2P Proxy] Отсылаю запрос к пиру", bestPeer.ID.String())
	//
	//	s.manager.UpdateRequest(bestPeer.ID)
	//
	//	resp, err := s.sendToPeer(bestPeer.ID, env)
	//	if err != nil {
	//		continue
	//	}
	//
	//	if resp.Type == "error" {
	//		fmt.Printf("[P2P Proxy] Удаленный пир вернул ошибку: %s", string(resp.Data))
	//		continue
	//	}
	//
	//	if resp.Type == "busy" {
	//		log.Println("[P2P Proxy] Узел занят ищем другой", bestPeer.ID.String())
	//		continue
	//	}
	//
	//	return resp.Data, nil
	//}
	return nil, fmt.Errorf("All peers not respond")
}
