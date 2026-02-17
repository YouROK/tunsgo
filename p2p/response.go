package p2p

import (
	"encoding/json"
	"log"

	"github.com/libp2p/go-libp2p/core/network"
)

func (s *P2PServer) handleInboundStream(stream network.Stream) {
	defer stream.Close()

	var env envelope
	if err := json.NewDecoder(stream).Decode(&env); err != nil {
		log.Printf("Ошибка декодирования Envelope: %v", err)
		return
	}

	var result response

	switch env.Type {
	case "ping":
		result = s.handlePingCommand(stream)
	case "proxy":
		result = s.handleProxyCommand(env.Payload)

	default:
		result = response{Type: "error", Data: []byte("unknown command type")}
	}

	respBytes, _ := json.Marshal(result)
	stream.Write(respBytes)
}
