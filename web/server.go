package web

import (
	"TunsGo/config"
	"TunsGo/net"
	"TunsGo/web/static"
	"log"
	"strconv"

	"github.com/gin-gonic/gin"
)

type WebServer struct {
	r *gin.Engine
}

func NewServer() *WebServer {
	return &WebServer{}
}

func (ws *WebServer) Run(manager *net.Manager) {
	ws.r = gin.Default()

	ws.r.Use(UseManager(manager))

	static.RouteEmbedFiles(ws.r)
	ws.SetupRoutesPages()

	err := ws.r.Run(config.Cfg.Web.Host + ":" + strconv.Itoa(config.Cfg.Web.Port))
	if err != nil {
		log.Println("Error starting server:", err)
	}
}

func UseManager(m *net.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("manager", m)
		c.Next()
	}
}
