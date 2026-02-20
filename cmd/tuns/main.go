package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/yourok/tunsgo/opts"
	"github.com/yourok/tunsgo/p2p"
	"github.com/yourok/tunsgo/version"
	"gopkg.in/yaml.v3"
)

const defRendezvous = "tunsgo-peers-0009"

func main() {
	ProtocolIDPtr := flag.String("proto", version.Version, "Protocol ID for p2p")
	RendezvousStringPtr := flag.String("rendezvous", defRendezvous, "Rendezvous address")

	flag.Parse()

	opts := opts.DefOptions()
	buf, err := os.ReadFile("tuns.conf")
	if err != nil {
		buf, _ = yaml.Marshal(&opts)
		os.WriteFile("tuns.conf", buf, 0644)
	} else {
		err = yaml.Unmarshal(buf, &opts)
		if err != nil {
			log.Fatal(err)
		}
	}

	ProtocolID := "/tunsgo/" + *ProtocolIDPtr
	RendezvousString := *RendezvousStringPtr

	server, err := p2p.NewP2PServer(ProtocolID, RendezvousString, opts)
	if err != nil {
		log.Fatal(err)
	}

	gin.SetMode(gin.ReleaseMode)
	route := gin.New()

	route.Use(gin.Logger(), gin.Recovery())

	route.Any("/proxy/*url", server.GinHandler)
	route.GET("/status", func(c *gin.Context) {
		st := server.Status()
		c.JSON(http.StatusOK, st)
	})

	httpSrv := &http.Server{
		Addr:    ":" + opts.Server.Port,
		Handler: route,
	}

	go func() {
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTP server error: %s\n", err)
		}
	}()
	fmt.Println("HTTP сервер запущен на :8080")

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-sigc
	server.Stop()
}
