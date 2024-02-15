package web

import (
	"net/http"

	"github.com/O1MaGnUmO1/erinaceus-vrf/core/services/erinaceus"

	"github.com/gin-gonic/gin"
)

// PingController has the ping endpoint.
type PingController struct {
	App erinaceus.Application
}

// Show returns pong.
func (eic *PingController) Show(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "pong"})
}
