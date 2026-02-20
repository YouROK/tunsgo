package urlproxy

import (
	"context"
	"log"
	"maps"
	"net/http"
	"sync"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/yourok/tunsgo/opts"
	"github.com/yourok/tunsgo/p2p/models"
	"github.com/yourok/tunsgo/version"
)

type UrlProxy struct {
	host host.Host
	opts *opts.Options
	ctx  context.Context

	slots chan struct{}

	httpClient *http.Client

	ps    *pubsub.PubSub
	topic *pubsub.Topic

	peers   map[peer.ID]*models.PeerInfo
	muPeers sync.RWMutex
}

func NewUrlProxy(c *models.SrvCtx) *UrlProxy {
	return &UrlProxy{
		host:  c.Host,
		opts:  c.Opts,
		ctx:   c.Ctx,
		slots: c.Slots,
		peers: make(map[peer.ID]*models.PeerInfo),
	}
}

func (p *UrlProxy) Start() error {
	log.Println("[P2P] Starting URL Proxy")

	ps, err := pubsub.NewGossipSub(p.ctx, p.host)
	if err != nil {
		return err
	}
	p.ps = ps

	topic, err := ps.Join("tunsgo-urlproxy-pubsub-" + version.Version)
	if err != nil {
		return err
	}
	p.topic = topic

	p.httpClient = NewP2PClient(p.host, p.ProtocolID())

	if len(p.opts.Hosts) > 0 {
		go p.announceHosts()
	}
	go p.listenForAnnouncements()
	go p.cleanupPeersGC()

	return nil
}

func (p *UrlProxy) Stop() {
	log.Println("[P2P] Stopping URL Proxy")
	p.topic.Close()
}

func (p *UrlProxy) Name() string {
	return "URLProxy"
}

func (p *UrlProxy) ProtocolID() protocol.ID {
	return "/tunsgo/urlproxy/1.0.0"
}

func (p *UrlProxy) GetPeers() map[peer.ID]*models.PeerInfo {
	p.muPeers.RLock()
	defer p.muPeers.RUnlock()

	return maps.Clone(p.peers)
}
