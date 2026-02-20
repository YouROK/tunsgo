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

	select {
	case s.slots <- struct{}{}:
		defer func() {
			go func() {
				time.Sleep(time.Duration(s.opts.Server.SlotSleep) * time.Second)
				<-s.slots
			}()
		}()
	default:
		json.NewEncoder(stream).Encode(P2PProxyResponse{StatusCode: 600, Body: []byte("no slots")})
		return
	}

	pid := stream.Conn().RemotePeer().String()

	var p2pReq *P2PProxyRequest
	decoder := json.NewDecoder(stream)
	if err := decoder.Decode(&p2pReq); err != nil {
		log.Printf("[P2P-Proxy] Request decoding error: %v", err)
		json.NewEncoder(stream).Encode(P2PProxyResponse{StatusCode: 601, Body: []byte(err.Error())})
		return
	}

	log.Printf("[P2P-Proxy] Inbound request from %s: %v", pid, p2pReq.URL)

	p2pResp, err, code := sendLocalRequest(s.httpClient, p2pReq, s.opts.Hosts)
	if code == 602 {
		log.Printf("[P2P-Proxy] Blocked: host %s is not whitelisted", p2pReq.URL)
		json.NewEncoder(stream).Encode(P2PProxyResponse{StatusCode: 602, Body: []byte("not support this host")})
		return
	}
	if code == 603 {
		log.Printf("[P2P-Proxy] Error creating HTTP request: %v", err)
		json.NewEncoder(stream).Encode(P2PProxyResponse{StatusCode: 603, Body: []byte(err.Error())})
		return
	}
	if code == 604 {
		log.Printf("[P2P-Proxy] Error receiving HTTP request: %v", err)
		json.NewEncoder(stream).Encode(P2PProxyResponse{StatusCode: 604, Body: []byte(err.Error())})
		return
	}

	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(p2pResp); err != nil {
		log.Printf("[P2P-Proxy] Error sending response to node %s: %v", pid, err)
	}
}

func sendLocalRequest(cli *http.Client, req *P2PProxyRequest, hosts []string) (*P2PProxyResponse, error, int) {
	u, err := url.Parse(req.URL)
	if err != nil || !matchHost(hosts, u.Host) {
		return nil, err, 602
	}

	httpReq, err := http.NewRequest(req.Method, req.URL, bytes.NewReader(req.Body))
	if err != nil {
		return nil, err, 603
	}

	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := cli.Do(httpReq)
	if err != nil {
		return nil, err, 604
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	p2pResp := &P2PProxyResponse{
		StatusCode: resp.StatusCode,
		Headers:    make(map[string]string),
		Body:       body,
	}

	for k, v := range resp.Header {
		p2pResp.Headers[k] = v[0]
	}

	return p2pResp, nil, 0
}
