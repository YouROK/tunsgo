package p2p

import (
	"context"
	"gotuns/config"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	tls "github.com/libp2p/go-libp2p/p2p/security/tls"
	"github.com/multiformats/go-multihash"
)

type P2PServer struct {
	host       host.Host
	dht        *dht.IpfsDHT
	ctx        context.Context
	cm         *connmgr.BasicConnMgr
	cId        cid.Cid
	protocolID protocol.ID

	slots   []time.Time
	muSlots sync.RWMutex

	ps    *pubsub.PubSub
	topic *pubsub.Topic

	cfg *config.Config

	peers   map[peer.ID]*peerInfo
	muPeers sync.RWMutex

	httpClient *http.Client
}

func NewP2PServer(cfg *config.Config, protocolID, rendezvous string) (*P2PServer, error) {
	key, err := LoadOrCreateIdentity()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	cm, err := connmgr.NewConnManager(
		cfg.P2P.LowConns,
		cfg.P2P.HiConns,
		connmgr.WithGracePeriod(time.Second*30),
	)
	if err != nil {
		return nil, err
	}

	opts := []libp2p.Option{
		libp2p.Identity(key),
		libp2p.ChainOptions(libp2p.DefaultPrivateTransports),
		libp2p.Security(tls.ID, tls.New),
		libp2p.ListenAddrStrings(
			"/ip4/0.0.0.0/tcp/0",
			//"/ip4/0.0.0.0/udp/0/quic-v1",
			"/ip4/0.0.0.0/udp/0/webrtc-direct",
			//"/ip4/0.0.0.0/tcp/443/wss",
		),
		libp2p.ConnectionManager(cm),
		libp2p.NATPortMap(),

		libp2p.EnableRelay(),
		libp2p.EnableRelayService(),
		libp2p.EnableNATService(),
		libp2p.EnableHolePunching(),
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

	pref := cid.Prefix{
		Version:  1,
		Codec:    cid.Raw,
		MhType:   multihash.SHA2_256,
		MhLength: -1,
	}
	c, _ := pref.Sum([]byte(rendezvous))

	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		return nil, err
	}

	topic, err := ps.Join("pubsub-" + rendezvous)
	if err != nil {
		return nil, err
	}

	srv := &P2PServer{
		host:       h,
		dht:        idht,
		ctx:        ctx,
		cm:         cm,
		protocolID: protocol.ID(protocolID),
		cfg:        cfg,
		cId:        c,
		ps:         ps,
		topic:      topic,
		peers:      make(map[peer.ID]*peerInfo),
		httpClient: &http.Client{Timeout: 20 * time.Second},
		slots:      make([]time.Time, cfg.Server.Slots),
	}

	h.SetStreamHandler(srv.protocolID, srv.handleInboundStream)

	go srv.startDiscovery()

	log.Println("[P2P] ID", h.ID().String())

	return srv, nil
}

func (s *P2PServer) Stop() {
	log.Println("[P2P] Остановка сервера...")
	s.topic.Close()
	s.dht.Close()
	s.host.Close()
	s.cm.Close()
}
