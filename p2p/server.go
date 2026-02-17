package p2p

import (
	"context"
	"gotuns/config"
	"gotuns/version"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	"github.com/multiformats/go-multiaddr"
)

const (
	ProtocolID       = "/tunsgo/" + version.Version
	RendezvousString = "tunsgo-discovery-0004"
)

type P2PServer struct {
	host    host.Host
	manager *PeerManager
	dht     *dht.IpfsDHT
	ctx     context.Context
	cm      *connmgr.BasicConnMgr
	slots   []time.Time

	muSlots sync.Mutex

	cfg *config.Config
}

func NewP2PServer(cfg *config.Config) (*P2PServer, error) {
	key, err := LoadOrCreateIdentity()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	cm, err := connmgr.NewConnManager(
		cfg.P2P.LowConns,
		cfg.P2P.HiConns,
		connmgr.WithGracePeriod(time.Minute),
	)
	if err != nil {
		return nil, err
	}

	savedRelays := loadRelays()

	opts := []libp2p.Option{
		libp2p.Identity(key),
		libp2p.ListenAddrStrings(
			"/ip4/0.0.0.0/tcp/0",
			"/ip4/0.0.0.0/udp/0/quic-v1",
		),
		libp2p.ConnectionManager(cm),
		libp2p.NATPortMap(),
		libp2p.EnableRelay(),
		libp2p.EnableHolePunching(),
	}
	if len(savedRelays) > 0 {
		opts = append(opts, libp2p.EnableAutoRelayWithStaticRelays(savedRelays))
	}

	if cfg.P2P.IsRelay {
		opts = append(opts, libp2p.EnableRelayService())
		log.Println("[P2P] Режим Relay Service включен")
	}

	h, err := libp2p.New(opts...)
	if err != nil {
		cm.Close()
		return nil, err
	}

	idht, err := dht.New(ctx, h, dht.Mode(dht.ModeAutoServer))
	if err != nil {
		cm.Close()
		return nil, err
	}

	srv := &P2PServer{
		host:    h,
		manager: NewPeerManager(),
		dht:     idht,
		ctx:     ctx,
		cm:      cm,
		slots:   make([]time.Time, cfg.Server.Slots),
		cfg:     cfg,
	}

	h.SetStreamHandler(ProtocolID, srv.handleInboundStream)

	return srv, nil
}

func (s *P2PServer) Stop() {
	log.Println("[P2P] Остановка сервера...")
	s.dht.Close()
	s.host.Close()
	s.cm.Close()
}

func loadRelays() []peer.AddrInfo {
	var infos []peer.AddrInfo
	bytes, err := os.ReadFile("relays.list")
	if err != nil {
		return infos
	}

	list := strings.Split(string(bytes), "\n")

	for _, sAddr := range list {
		ma, err := multiaddr.NewMultiaddr(sAddr)
		if err != nil {
			continue
		}
		pi, err := peer.AddrInfoFromP2pAddr(ma)
		if err != nil {
			continue
		}
		infos = append(infos, *pi)
	}
	return infos
}
