package p2p

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	tls "github.com/libp2p/go-libp2p/p2p/security/tls"
	"github.com/multiformats/go-multihash"
	"github.com/yourok/tunsgo/opts"
	"github.com/yourok/tunsgo/p2p/models"
	"github.com/yourok/tunsgo/version"
)

type P2PServer struct {
	host       host.Host
	dht        *dht.IpfsDHT
	ctx        context.Context
	cm         *connmgr.BasicConnMgr
	cId        cid.Cid
	protocolID protocol.ID

	slots chan struct{}

	ps    *pubsub.PubSub
	topic *pubsub.Topic

	opts *opts.Options

	peers   map[peer.ID]*models.PeerInfo
	muPeers sync.RWMutex

	httpClient *http.Client
}

func NewP2PServer(protocolID, rendezvous string, opts *opts.Options) (*P2PServer, error) {
	log.Println("[P2P Server] Starting...")
	log.Println("[P2P Server] Version:", version.Version)
	log.Println("[P2P Server] Provide hosts:", opts.Hosts)
	log.Println("[P2P Server] ProtocolID:", protocolID)
	log.Println("[P2P Server] Rendezvous:", rendezvous)

	key, err := LoadOrCreateIdentity()
	if err != nil {
		return nil, err
	}

	if opts.Server.Slots < 1 { //min 1 slot
		opts.Server.Slots = 1
	}
	if opts.Server.SlotSleep < 0 { // min sleep 0
		opts.Server.SlotSleep = 0
	}
	if opts.Server.SlotSleep > 300 { //max sleep 5 min
		opts.Server.SlotSleep = 0
	}

	ctx := context.Background()

	cm, err := connmgr.NewConnManager(
		opts.P2P.LowConns,
		opts.P2P.HiConns,
		connmgr.WithGracePeriod(time.Second*30),
	)
	if err != nil {
		return nil, err
	}

	optsLp2p := []libp2p.Option{
		libp2p.Identity(key),
		libp2p.ChainOptions(libp2p.DefaultPrivateTransports),
		libp2p.Security(tls.ID, tls.New),
		libp2p.ListenAddrStrings(
			"/ip4/0.0.0.0/tcp/0",
			//"/ip4/0.0.0.0/udp/0/quic-v1",
			//"/ip4/0.0.0.0/udp/0/webrtc-direct",
			//"/ip4/0.0.0.0/tcp/443/wss",
		),
		libp2p.ConnectionManager(cm),
		libp2p.NATPortMap(),

		libp2p.EnableRelay(),
		libp2p.EnableRelayService(),
		libp2p.EnableNATService(),
		libp2p.EnableHolePunching(),
	}

	h, err := libp2p.New(optsLp2p...)
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
		opts:       opts,
		cId:        c,
		ps:         ps,
		topic:      topic,
		peers:      make(map[peer.ID]*models.PeerInfo),
		slots:      make(chan struct{}, opts.Server.Slots),
	}

	srv.httpClient = NewP2PClient(srv.host, srv.protocolID)

	h.SetStreamHandler(srv.protocolID, srv.handleInboundStream)

	go srv.startDiscovery()

	log.Println("[P2P] ID", h.ID().String())

	go srv.watchNetworkStatus()

	return srv, nil
}

func (s *P2PServer) Stop() {
	log.Println("[P2P Server] Stoping...")
	s.topic.Close()
	s.dht.Close()
	s.host.Close()
	s.cm.Close()
}

func (s *P2PServer) watchNetworkStatus() {
	sub, err := s.host.EventBus().Subscribe([]interface{}{
		new(event.EvtLocalReachabilityChanged),
	})
	if err != nil {
		log.Printf("[NET] Failed to subscribe to reachability events: %v", err)
		return
	}

	go func() {
		defer sub.Close()

		for {
			select {
			case e, ok := <-sub.Out():
				if !ok {
					log.Println("[NET] Event bus channel closed")
					return
				}

				evt := e.(event.EvtLocalReachabilityChanged)
				switch evt.Reachability {
				case network.ReachabilityPublic:
					log.Println("[NET] Reachability changed: PUBLIC. Node is now operating as a Relay Hop")
				case network.ReachabilityPrivate:
					log.Println("[NET] Reachability changed: PRIVATE. Node is operating behind NAT (client mode)")
				case network.ReachabilityUnknown:
					log.Println("[NET] Reachability changed: UNKNOWN. Determining network status...")
				}

			case <-s.ctx.Done():
				log.Println("[NET] Stopping network reachability")
				return
			}
		}
	}()
}
