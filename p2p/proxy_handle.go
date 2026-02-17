package p2p

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func (s *P2PServer) handleProxyCommand(payload []byte) response {
	slot := s.findFreeSlots()
	if slot < 0 {
		return response{
			Type: "busy",
			Data: []byte("no empty slots"),
		}
	}
	s.muSlots.Lock()
	s.slots[slot] = time.Now().Add(time.Minute)
	s.muSlots.Unlock()
	defer func() {
		s.muSlots.Lock()
		s.slots[slot] = time.Now().Add(time.Second * time.Duration(s.cfg.Server.SlotSleep))
		s.muSlots.Unlock()
	}()

	var cmd proxyCommand
	if err := json.Unmarshal(payload, &cmd); err != nil {
		return response{
			Type: "error",
			Data: []byte("invalid proxy command payload"),
		}
	}

	requrl, err := url.Parse(cmd.URL)
	if err != nil {
		return response{
			Type: "error",
			Data: []byte("invalid url"),
		}
	}

	allowed := s.isHostAllowed(requrl.Host)
	if !allowed {
		log.Printf("[P2P] Запрещенный хост: %s", cmd.URL)
		return response{
			Type: "error",
			Data: []byte("host is banned on proxy"),
		}
	}

	log.Printf("[P2P] Исполняю прокси-запрос к: %s", cmd.URL)

	client := http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(cmd.URL)
	if err != nil {
		log.Printf("[P2P] Ошибка HTTP запроса: %v", err)
		return response{
			Type: "error",
			Data: []byte(err.Error()),
		}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return response{
			Type: "error",
			Data: []byte("failed to read target response body"),
		}
	}

	log.Printf("[P2P] Запрос выполнен успешно, получено %d байт", len(body))

	return response{
		Type: "proxy",
		Data: body,
	}
}

func (s *P2PServer) findFreeSlots() int {
	s.muSlots.Lock()
	defer s.muSlots.Unlock()
	for i, slot := range s.slots {
		if slot.Before(time.Now()) {
			return i
		}
	}
	return -1
}

func inList(host string, list []string) bool {
	for _, s := range list {
		if strings.ToLower(host) == strings.ToLower(s) {
			return true
		}
	}
	return false
}

func (s *P2PServer) isHostAllowed(host string) bool {
	whiteLen := len(s.cfg.Hosts.Whitelist)
	blackLen := len(s.cfg.Hosts.Blacklist)

	if whiteLen > 0 {
		if inList(host, s.cfg.Hosts.Whitelist) {
			return true
		}
		return false
	}

	if blackLen > 0 {
		if inList(host, s.cfg.Hosts.Blacklist) {
			return false
		}
	}
	return true
}
