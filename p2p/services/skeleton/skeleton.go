package skeleton

import (
	"context"
	"log"

	"github.com/YouROK/tunsgo/opts"
	"github.com/YouROK/tunsgo/p2p/models"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/protocol"
)

type Skeleton struct {
	host host.Host
	opts *opts.Options
	ctx  context.Context
}

func NewSkeleton(c *models.SrvCtx) *Skeleton {
	return &Skeleton{
		host: c.Host,
		opts: c.Opts,
		ctx:  c.Ctx,
	}
}

func (p *Skeleton) Start() error {
	log.Println("[P2P] Starting skeleton")
	return nil
}

func (p *Skeleton) Stop() {
	log.Println("[P2P] Stopping URL Proxy")
}

func (p *Skeleton) Name() string {
	return "Skeleton"
}

func (p *Skeleton) ProtocolID() protocol.ID {
	return "/tunsgo/skeleton/1.0.0"
}

func (p *Skeleton) HandleStream(stream network.Stream) {
	defer stream.Close()
}
