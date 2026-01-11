package main

import (
	"TunsGo/config"
	"TunsGo/net"
	"TunsGo/web"
	"log"
	"os"
	"os/signal"
	"syscall"

	"gopkg.in/natefinch/lumberjack.v2"
)

func main() {
	log.Println("Starting TunsGo")
	err := config.Load()
	if err != nil {
		return
	}
	initLogger()

	m := net.NewManager()
	m.Start()
	server := web.NewServer()
	go server.Run(m)

	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	<-osSignals

	m.Stop()
}

func initLogger() {
	if !config.Cfg.Log.UseStdOut {
		lumberLog := &lumberjack.Logger{
			Filename:   config.Cfg.Log.Filename,
			MaxSize:    config.Cfg.Log.MaxSize,
			MaxBackups: config.Cfg.Log.MaxBackups,
			MaxAge:     config.Cfg.Log.MaxAge,
			Compress:   config.Cfg.Log.Compress,
		}

		log.SetOutput(lumberLog)
		log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	}
}
