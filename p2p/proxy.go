package p2p

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

type P2PProxyRequest struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    []byte            `json:"body,omitempty"`
}

type P2PProxyResponse struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       []byte            `json:"body"`
}

func (s *P2PServer) GinHandler(c *gin.Context) {
	body, _ := io.ReadAll(c.Request.Body)

	headers := make(map[string]string)
	for k, v := range c.Request.Header {
		headers[k] = v[0]
	}

	p2pReq := &P2PProxyRequest{
		Method:  c.Request.Method,
		URL:     c.Query("url"),
		Headers: headers,
		Body:    body,
	}

	p2pResp, err, code := sendLocalRequest(s.httpClient, p2pReq, s.opts.Hosts)
	if err != nil || code != 0 {
		p2pResp, err = s.SendRequestProxyP2P(p2pReq)
		if err != nil {
			c.JSON(502, gin.H{"error": "P2P Proxy failed", "details": err.Error()})
			return
		}
	}
	for k, v := range p2pResp.Headers {
		c.Header(k, v)
	}
	c.Data(p2pResp.StatusCode, c.Writer.Header().Get("Content-Type"), p2pResp.Body)
}

func (s *P2PServer) SendRequestProxyP2P(reqData *P2PProxyRequest) (*P2PProxyResponse, error) {
	u, err := url.Parse(reqData.URL)
	if err != nil {
		return nil, err
	}

	candidates := s.getCandidateProxies(u.Host)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no proxy nodes for %s", u.Host)
	}

	var lastErr error
	for _, pID := range candidates {
		if s.host.Network().Connectedness(pID) != network.Connected {
			ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
			err := s.host.Connect(ctx, peer.AddrInfo{ID: pID})
			cancel()
			if err != nil {
				continue
			}
		}

		resp, err := s.doStreamRequest(pID, reqData)

		s.muPeers.Lock()
		if info, ok := s.peers[pID]; ok {
			info.LastResp = time.Now()
		}
		s.muPeers.Unlock()

		if err != nil {
			lastErr = err
			log.Printf("[P2P] Ошибка от пира %s: %v", pID, err)
			s.host.ConnManager().UpsertTag(pID, "tuns-node", func(current int) int {
				if current > 10 {
					return current - 10
				}
				return 0
			})
			continue
		}

		if resp.StatusCode == 600 {
			log.Printf("[P2P] Peer has no free slots %s", pID)
			s.host.ConnManager().UpsertTag(pID, "tuns-node", func(current int) int {
				if current > 20 {
					return current - 20
				}
				return 0
			})
			continue
		}

		if resp.StatusCode > 600 {
			log.Printf("[P2P] Error executing request %s: %v", pID, string(resp.Body))
			s.host.ConnManager().UpsertTag(pID, "tuns-node", func(current int) int {
				if current > 10 {
					return current - 10
				}
				return 0
			})
			continue
		}

		s.host.ConnManager().UpsertTag(pID, "tuns-node", func(current int) int {
			if current < 90 {
				return current + 10
			}
			return 100
		})
		return resp, nil
	}

	return nil, fmt.Errorf("all candidates failed. Last error: %v", lastErr)
}

func (s *P2PServer) doStreamRequest(pID peer.ID, reqData *P2PProxyRequest) (*P2PProxyResponse, error) {
	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	log.Println("[P2Proxy] Request to", pID.String(), "Url:", reqData.Method, reqData.URL)

	stream, err := s.host.NewStream(ctx, pID, s.protocolID)
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	if err := json.NewEncoder(stream).Encode(reqData); err != nil {
		return nil, err
	}

	stream.CloseWrite()

	var p2pResp P2PProxyResponse
	if err := json.NewDecoder(stream).Decode(&p2pResp); err != nil {
		return nil, err
	}

	return &p2pResp, nil
}

func (s *P2PServer) getCandidateProxies(targetHost string) []peer.ID {
	s.muPeers.RLock()
	defer s.muPeers.RUnlock()

	type candidate struct {
		id   peer.ID
		last time.Time
	}
	var list []candidate

	for pID, info := range s.peers {
		if matchHost(info.Hosts, targetHost) {
			list = append(list, candidate{pID, info.LastResp})
		}
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].last.Before(list[j].last)
	})

	result := make([]peer.ID, len(list))
	for i, c := range list {
		result[i] = c.id
	}
	return result
}

func matchHost(patterns []string, target string) bool {
	target = strings.ToLower(target)
	for _, pattern := range patterns {
		pattern = strings.ToLower(pattern)

		if pattern == target || pattern == "*" {
			return true
		}

		if strings.HasPrefix(pattern, "*") {
			suffix := pattern[1:]
			if strings.HasPrefix(pattern, ".") {
				suffix = suffix[1:]
			}
			if strings.HasSuffix(target, suffix) {
				return true
			}
		}
	}
	return false
}
