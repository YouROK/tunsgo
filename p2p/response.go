package p2p

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/yourok/tunsgo/p2p/utils"
)

func (s *P2PServer) handleInboundStream(stream network.Stream) {
	defer stream.Close()

	select {
	case s.slots <- struct{}{}:
		defer func() {
			go func() {
				time.Sleep(time.Duration(s.opts.Server.SlotSleep) * time.Second)
				<-s.slots
			}()
		}()
	default:
		fmt.Fprintf(stream, "HTTP/1.1 429 Too Many Requests\r\n\r\nAll slots busy")
		return
	}

	reader := bufio.NewReader(stream)

	line, err := reader.ReadString('\n')
	if err != nil {
		return
	}

	targetAddr := strings.TrimPrefix(strings.TrimSpace(line), "CONNECT ")

	hostOnly, _, _ := net.SplitHostPort(targetAddr)
	if hostOnly == "" {
		hostOnly = targetAddr
	}

	if !utils.MatchHost(s.opts.Hosts, hostOnly) {
		fmt.Fprintf(stream, "HTTP/1.1 403 Forbidden\r\n\r\nHost Not Allowed")
		return
	}

	conn, err := net.DialTimeout("tcp", targetAddr, 15*time.Second)
	if err != nil {
		fmt.Fprintf(stream, "HTTP/1.1 502 Bad Gateway\r\n\r\nFailed to connect to target: %v", err)
		return
	}

	errChan := make(chan error, 2)
	go func() {
		_, err := io.Copy(conn, reader)
		errChan <- err
	}()

	go func() {
		_, err := io.Copy(stream, conn)
		errChan <- err
	}()

	<-errChan
	conn.Close()
	<-errChan
}
