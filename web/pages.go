package web

import (
	"TunsGo/config"
	"TunsGo/net"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func IndexPage(c *gin.Context) {
	val, _ := c.Get("manager")
	mgr := val.(*net.Manager)

	status, err := mgr.GetSystemStatus()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.HTML(http.StatusOK, "index.go.html", gin.H{
		"Status": status,
		"Config": config.Cfg,
	})
}

func ConfigPage(c *gin.Context) {
	c.HTML(http.StatusOK, "config_edit.go.html", config.Cfg)
}

func ConfigSaveHandler(c *gin.Context) {
	// 1. Общие настройки
	config.Cfg.FillRouteTable = c.PostForm("fill_route_table") == "on"

	// 2. DNS
	config.Cfg.DNS.Listen = c.PostForm("dns_listen")
	config.Cfg.DNS.Upstream = c.PostForm("dns_upstream")
	config.Cfg.DNS.ForwardTun = c.PostForm("dns_forward_tun")
	config.Cfg.DNS.UseCache = c.PostForm("dns_use_cache") == "on"

	// 3. Web
	config.Cfg.Web.Host = c.PostForm("web_host")
	if port, err := strconv.Atoi(c.PostForm("web_port")); err == nil {
		config.Cfg.Web.Port = port
	}
	config.Cfg.Web.Login = c.PostForm("web_login")

	// Пароль обновляем только если поле не пустое
	newPass := c.PostForm("web_pass")
	if newPass != "" {
		config.Cfg.Web.Pass = newPass
	}

	// 4. Log
	config.Cfg.Log.Filename = c.PostForm("log_filename")
	config.Cfg.Log.UseStdOut = c.PostForm("log_stdout") == "on"
	if maxSize, err := strconv.Atoi(c.PostForm("log_maxsize")); err == nil {
		config.Cfg.Log.MaxSize = maxSize
	}

	// 5. Сохранение в файл через ваш метод config.Save()
	if err := config.Save(); err != nil {
		c.String(http.StatusInternalServerError, "Failed to save config: %s", err.Error())
		return
	}

	// Редирект обратно на страницу настроек с флагом успеха
	c.Redirect(http.StatusSeeOther, "/config?success=true")
}
