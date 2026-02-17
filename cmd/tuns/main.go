package main

import (
	"fmt"
	"gotuns/config"
	"gotuns/p2p"
	"log"
	"net/http"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	server, err := p2p.NewP2PServer(cfg)
	if err != nil {
		log.Fatal(err)
	}

	server.StartDiscovery()

	http.HandleFunc("/proxy", func(w http.ResponseWriter, r *http.Request) {
		targetURL := r.URL.Query().Get("url")

		if targetURL == "" {
			http.Error(w, "Missing 'url' parameter. Usage: /proxy?url=http://...", http.StatusBadRequest)
			return
		}

		log.Printf("[HTTP] Получен запрос на проксирование: %s", targetURL)

		buf, err := server.SendRequestProxy(targetURL)
		if err != nil {
			log.Printf("[HTTP] Ошибка P2P: %v", err)
			http.Error(w, "P2P error: "+err.Error(), http.StatusBadGateway)
			return
		}

		contentType := http.DetectContentType(buf)
		w.Header().Set("Content-Type", contentType)

		w.Write(buf)
	})

	fmt.Println("HTTP сервер запущен на :8080")
	http.ListenAndServe(":8080", nil)

	select {}
}
