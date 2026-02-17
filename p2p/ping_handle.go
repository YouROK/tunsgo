package p2p

import (
	"encoding/json"
	"gotuns/version"
	"log"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
)

func (s *P2PServer) handlePingCommand(stream network.Stream) response {
	hData, _ := json.Marshal(handshakeData{
		Version: version.Version,
	})

	rid := stream.Conn().RemotePeer()
	if !s.manager.Exist(rid) {
		go func() {
			//Ожидание пока нас добавят
			time.Sleep(time.Second)
			log.Printf("[P2P] Узел %s добавил нас, ping его в ответ", rid.String())
			s.pingCmd(rid)
		}()
	}

	return response{Type: "ping", Data: hData}
}
