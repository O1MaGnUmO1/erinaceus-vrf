package web

import (
	"net/http"

	"github.com/O1MaGnUmO1/erinaceus-vrf/core/services/erinaceus"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/static"

	"github.com/gin-gonic/gin"
)

// BuildVersonController has the build_info endpoint.
type BuildInfoController struct {
	App erinaceus.Application
}

// Show returns the build info.
func (eic *BuildInfoController) Show(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"version": static.Version, "commitSHA": static.Sha})
}
