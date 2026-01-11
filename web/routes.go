package web

import (
	"TunsGo/config"

	"github.com/gin-gonic/gin"
)

func getAuth() gin.HandlerFunc {
	if config.Cfg.Web.Login != "" && config.Cfg.Web.Pass != "" {
		return gin.BasicAuth(gin.Accounts{
			config.Cfg.Web.Login: config.Cfg.Web.Pass,
		})
	}
	return nil
}

func (ws *WebServer) SetupRoutesPages() {
	//Page routes
	auth := ws.r.Group("/", getAuth())

	auth.GET("/", IndexPage)
	auth.GET("/config", ConfigPage)
	auth.POST("/config/save", ConfigSaveHandler)
}
