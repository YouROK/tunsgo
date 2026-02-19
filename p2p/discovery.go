package p2p

import (
	"context"
	"log"
	"sync"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

func (s *P2PServer) startDiscovery() {
	time.Sleep(time.Second * 5)
	s.bootstrap()
	go s.announceDht()
	go s.discoveryPeers()
	if len(s.cfg.Hosts.ProvidedHosts) > 0 {
		go s.announceHosts()
	}
	go s.listenForAnnouncements()
}

func (s *P2PServer) bootstrap() {
	successConnections := 0

	var wa sync.WaitGroup
	wa.Add(len(dht.DefaultBootstrapPeers))

	go func() {
		log.Println("[P2P] Подключение к bootstrap")
		for _, addrStr := range dht.DefaultBootstrapPeers {
			addr, _ := multiaddr.NewMultiaddr(addrStr.String())
			pi, _ := peer.AddrInfoFromP2pAddr(addr)

			ctx, cancel := context.WithTimeout(s.ctx, 120*time.Second)
			if err := s.host.Connect(ctx, *pi); err == nil {
				successConnections++
			} else {
				log.Printf("[P2P] Ошибка подключения к bootstrap-узлу %s: %v", pi.ID, err)
			}
			cancel()
			wa.Done()
		}
	}()

	wa.Wait()

	if successConnections > 0 {
		log.Printf("[P2P] Успешно подключено к %d бутстрап-узлам", successConnections)
	} else {
		log.Println("[P2P] Предупреждение: не удалось подключиться к стандартным бутстрапам")
	}
}

func (s *P2PServer) announceDht() {

	if err := s.dht.Bootstrap(s.ctx); err != nil {
		log.Printf("[P2P] Ошибка DHT Bootstrap: %v", err)
	}

	ticker := time.NewTicker(20 * time.Minute)
	defer ticker.Stop()
	for {
		log.Println("[P2P] Анонс в DHT")
		ctx, cancel := context.WithTimeout(s.ctx, 5*time.Minute)
		err := s.dht.Provide(ctx, s.cId, true)
		cancel()
		if err != nil {
			log.Printf("[P2P] DHT Provide: %v", err)
			time.Sleep(20 * time.Second)
			continue
		} else {
			log.Println("[P2P] В DHT анонсирован")
		}

		select {
		case <-ticker.C:
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *P2PServer) discoveryPeers() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		ctx, cancel := context.WithTimeout(s.ctx, 1*time.Minute)
		peers := s.dht.FindProvidersAsync(ctx, s.cId, min(s.cfg.P2P.LowConns, 20))

		for p := range peers {
			if p.ID == s.host.ID() || len(p.Addrs) == 0 {
				continue
			}

			if s.host.Network().Connectedness(p.ID) != network.Connected {
				log.Println("[P2P] Соединение с узлом:", p.ID)
				err := s.host.Connect(s.ctx, p)
				if err == nil {
					log.Printf("[P2P] Подключился: %s", p.ID)
				} else {
					log.Printf("[P2P] Ошибка соединения с узлом %s", p.ID)
				}
			}
		}
		cancel()

		select {
		case <-ticker.C:
		case <-s.ctx.Done():
			return
		}
	}
}
