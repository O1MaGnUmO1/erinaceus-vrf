package web

import (
	"github.com/gin-gonic/gin"

	"github.com/O1MaGnUmO1/erinaceus-vrf/core/services/erinaceus"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/web/presenters"
)

// FeaturesController manages the feature flags
type FeaturesController struct {
	App erinaceus.Application
}

const (
	FeatureKeyCSA string = "csa"
)

// Index retrieves the features
// Example:
// "GET <application>/features"
func (fc *FeaturesController) Index(c *gin.Context) {
	resources := []presenters.FeatureResource{
		*presenters.NewFeatureResource(FeatureKeyCSA, fc.App.GetConfig().Feature().UICSAKeys()),
	}

	jsonAPIResponse(c, resources, "features")
}
