package main

import (
	"TunsGo/config"
	"TunsGo/net"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	log.Println("Starting TunsGo")
	err := config.Load()
	if err != nil {
		return
	}

	m := net.NewManager()
	m.Start()

	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, os.Interrupt, syscall.SIGTERM)
	<-osSignals

	m.Stop()
}
