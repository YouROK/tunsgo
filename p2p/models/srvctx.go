package models

import (
	"context"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/yourok/tunsgo/opts"
)

type SrvCtx struct {
	Host       host.Host
	Opts       *opts.Options
	Ctx        context.Context
	ProtocolID protocol.ID
	Slots      chan struct{}
}
