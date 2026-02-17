package p2p

import (
	"context"
	"fmt"
	"log"
	"os"
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
	s.bootstrap()
	go s.announceDht()
	if !s.cfg.P2P.IsRelay {
		go s.discoveryPeers()
	}
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
	// wait for bootstrap
	time.Sleep(5 * time.Second)
}

func (s *P2PServer) announceDht() {
	c := strToCid(RendezvousString)
	ticker := time.NewTicker(20 * time.Minute)
	defer ticker.Stop()
	for {
		log.Println("[P2P] Анонс в DHT")
		provideCtx, cancelProvide := context.WithTimeout(s.ctx, 30*time.Second)
		err := s.dht.Provide(provideCtx, c, true)
		cancelProvide()
		if err != nil {
			log.Printf("[P2P] DHT Provide: %v", err)
			time.Sleep(20 * time.Second)
			continue
		}

		select {
		case <-ticker.C:
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *P2PServer) discoveryPeers() {
	c := strToCid(RendezvousString)
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		log.Println("[P2P] Поиск узлов")

		findCtx, cancelFind := context.WithTimeout(s.ctx, 20*time.Second)
		peers := s.dht.FindProvidersAsync(findCtx, c, 20)

		for p := range peers {
			if p.ID == s.host.ID() || len(p.Addrs) == 0 {
				continue
			}

			log.Println("[P2P] Найден узел:", p.ID)
			if s.host.Network().Connectedness(p.ID) != network.Connected {
				log.Println("[P2P] Соединение с узлом:", p.ID)
				err := s.host.Connect(s.ctx, p)
				if err == nil {
					if !s.manager.Exist(p.ID) {
						log.Printf("[P2P] Подключился: %s", p.ID)
						go s.pingCmd(p.ID)
					}
				}
			}
		}
		cancelFind()

		if s.manager.Peers() == 0 {
			time.Sleep(5 * time.Second)
			continue
		}

		s.saveRelays()

		select {
		case <-ticker.C:
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *P2PServer) saveRelays() {
	list := s.manager.GetRelays()
	if len(list) == 0 {
		return
	}

	rawlist := ""

	for _, info := range list {
		addrs := s.host.Peerstore().Addrs(info.ID)
		for _, addr := range addrs {
			fullAddr := fmt.Sprintf("%s/p2p/%s\n", addr.String(), info.ID.String())
			rawlist += fullAddr
		}
	}

	os.WriteFile("relays.list", []byte(rawlist), 0644)
	log.Println("[P2P] Список relays сохранен в relays.list")
}
