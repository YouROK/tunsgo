package p2p

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
)

func (s *P2PServer) handleInboundStream(stream network.Stream) {
	defer stream.Close()

	slot := s.findFreeSlots()
	if slot < 0 {
		json.NewEncoder(stream).Encode(P2PProxyResponse{StatusCode: 600, Body: []byte("no slots")})
	}
	s.muSlots.Lock()
	s.slots[slot] = time.Now().Add(time.Minute)
	s.muSlots.Unlock()
	defer func() {
		s.muSlots.Lock()
		s.slots[slot] = time.Now().Add(time.Second * time.Duration(s.cfg.Server.SlotSleep))
		s.muSlots.Unlock()
	}()

	var p2pReq P2PProxyRequest
	decoder := json.NewDecoder(stream)
	if err := decoder.Decode(&p2pReq); err != nil {
		log.Printf("[P2P-Proxy] Ошибка декодирования запроса: %v", err)
		json.NewEncoder(stream).Encode(P2PProxyResponse{StatusCode: 601, Body: []byte("Error: " + err.Error())})
		return
	}

	u, err := url.Parse(p2pReq.URL)
	if err != nil || !matchHost(s.cfg.Hosts.ProvidedHosts, u.Host) {
		log.Printf("[P2P-Proxy] Блокировка: хост %s не в белом списке", p2pReq.URL)
		json.NewEncoder(stream).Encode(P2PProxyResponse{StatusCode: 602, Body: []byte("not support this host")})
		return
	}

	httpReq, err := http.NewRequest(p2pReq.Method, p2pReq.URL, bytes.NewReader(p2pReq.Body))
	if err != nil {
		log.Printf("[P2P-Proxy] Ошибка создания HTTP запроса: %v", err)
		json.NewEncoder(stream).Encode(P2PProxyResponse{StatusCode: 603, Body: []byte("Error: " + err.Error())})
		return
	}

	for k, v := range p2pReq.Headers {
		httpReq.Header.Set(k, v)
	}
	httpReq.Header.Set("User-Agent", "Mozilla/5.0 (P2P Proxy Node)")

	resp, err := s.httpClient.Do(httpReq)

	p2pResp := P2PProxyResponse{}

	if err != nil {
		log.Printf("[P2P-Proxy] Ошибка выполнения HTTP: %v", err)
		p2pResp.StatusCode = 603
		p2pResp.Body = []byte("Error: " + err.Error())
	} else {
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		p2pResp.StatusCode = resp.StatusCode
		p2pResp.Body = body
		p2pResp.Headers = make(map[string]string)

		for k, v := range resp.Header {
			p2pResp.Headers[k] = v[0]
		}
	}

	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(p2pResp); err != nil {
		log.Printf("[P2P-Proxy] Ошибка отправки ответа в стрим: %v", err)
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
