package resolver

import "github.com/O1MaGnUmO1/erinaceus-vrf/core/config"

type FeaturesResolver struct {
	cfg config.Feature
}

func NewFeaturesResolver(cfg config.Feature) *FeaturesResolver {
	return &FeaturesResolver{cfg: cfg}
}

// CSA resolves to whether CSA Keys are enabled
func (r *FeaturesResolver) CSA() bool {
	return r.cfg.UICSAKeys()
}

type FeaturesPayloadResolver struct {
	cfg config.Feature
}

func NewFeaturesPayloadResolver(cfg config.Feature) *FeaturesPayloadResolver {
	return &FeaturesPayloadResolver{cfg: cfg}
}

func (r *FeaturesPayloadResolver) ToFeatures() (*FeaturesResolver, bool) {
	return NewFeaturesResolver(r.cfg), true
}
