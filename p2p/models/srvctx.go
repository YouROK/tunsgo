package models

import (
	"context"
	"sync"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/yourok/tunsgo/opts"
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
