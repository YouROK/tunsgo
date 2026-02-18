package cmds

import (
	"encoding/json"
	"gotuns/version"

	"github.com/libp2p/go-libp2p/core/network"
)

func (s *P2PServer) handlePingCommand(stream network.Stream) response {
	hData, _ := json.Marshal(handshakeData{
		Version:       version.Version,
		ProvidedHosts: s.cfg.Hosts.ProvidedHosts,
	})

	//rid := stream.Conn().RemotePeer()
	//if !s.manager.Exist(rid) {
	//	go func() {
	//		time.Sleep(time.Second)
	//		s.pingCmd(rid)
	//	}()
	//}

	return response{Type: "ping", Data: hData}
}
