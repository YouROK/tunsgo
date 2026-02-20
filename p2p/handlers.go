package p2p

import "github.com/gin-gonic/gin"

func (s *P2PServer) GinHandler(c *gin.Context) {
	s.urlprx.GinHandler(c)
}
