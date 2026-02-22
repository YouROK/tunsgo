package models

import (
	"context"
	"sync"

	"github.com/YouROK/tunsgo/opts"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

type SrvCtx struct {
	Host       host.Host
	Opts       *opts.Options
	Ctx        context.Context
	ProtocolID protocol.ID
	Slots      chan struct{}

	Peers   map[peer.ID]*PeerInfo
	MuPeers sync.RWMutex
}
