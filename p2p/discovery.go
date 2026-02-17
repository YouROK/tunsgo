package p2p

import (
	"context"
	"log"
	"time"

	"github.com/ipfs/go-cid"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multihash"
)

func strToCid(s string) cid.Cid {
	pref := cid.Prefix{
		Version:  1,
		Codec:    cid.Raw,
		MhType:   multihash.SHA2_256,
		MhLength: -1,
	}
	c, _ := pref.Sum([]byte(s))
	return c
}

func (s *P2PServer) StartDiscovery() {
	go s.discoveryLoop()
}

func (s *P2PServer) bootstrap() {
	log.Println("[P2P] Подключение к глобальной сети...")

	successConnections := 0

	for _, addrStr := range dht.DefaultBootstrapPeers {
		addr, _ := multiaddr.NewMultiaddr(addrStr.String())
		pi, _ := peer.AddrInfoFromP2pAddr(addr)

		ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
		if err := s.host.Connect(ctx, *pi); err == nil {
			successConnections++
		}
		cancel()
	}

	if successConnections > 0 {
		log.Printf("[P2P] Успешно подключено к %d бутстрап-узлам", successConnections)
	} else {
		log.Println("[P2P] Предупреждение: не удалось подключиться к стандартным бутстрапам")
	}

	if err := s.dht.Bootstrap(s.ctx); err != nil {
		log.Printf("[P2P] Ошибка DHT Bootstrap: %v", err)
	}
}

func (s *P2PServer) discoveryLoop() {
	s.bootstrap()
	c := strToCid(RendezvousString)
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		provideCtx, cancelProvide := context.WithTimeout(s.ctx, 30*time.Second)
		if err := s.dht.Provide(provideCtx, c, true); err != nil {
			log.Printf("[P2P] DHT Provide: %v", err)
		}
		cancelProvide()

		findCtx, cancelFind := context.WithTimeout(s.ctx, 20*time.Second)
		peers := s.dht.FindProvidersAsync(findCtx, c, 20)

		for p := range peers {
			if p.ID == s.host.ID() || len(p.Addrs) == 0 {
				continue
			}

			if s.host.Network().Connectedness(p.ID) != network.Connected {
				err := s.host.Connect(s.ctx, p)
				if err == nil {
					if !s.manager.Exist(p.ID) {
						log.Printf("[P2P] Новый узел: %s", p.ID)
						go s.pingCmd(p.ID)
					}
				}
			}
		}
		cancelFind()

		select {
		case <-ticker.C:
		case <-s.ctx.Done():
			return
		}
	}
}
