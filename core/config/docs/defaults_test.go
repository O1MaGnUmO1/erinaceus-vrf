package docs

import (
	"testing"

	"github.com/O1MaGnUmO1/erinaceus-vrf/core/services/chainlink/cfgtest"
)

func TestCoreDefaults_notNil(t *testing.T) {
	cfgtest.AssertFieldsNotNil(t, CoreDefaults())
}
