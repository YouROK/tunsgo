package p2p

import (
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

type PeerInfo struct {
	ID      peer.ID
	Latency time.Duration
	LastReq time.Time
	Version string `json:"version"`
}
