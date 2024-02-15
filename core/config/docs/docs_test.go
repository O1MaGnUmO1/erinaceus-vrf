package docs_test

import (
	"strings"
	"testing"

	"github.com/kylelemons/godebug/diff"
	gotoml "github.com/pelletier/go-toml/v2"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/O1MaGnUmO1/chainlink-common/pkg/config"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/chains/evm/assets"
	evmcfg "github.com/O1MaGnUmO1/erinaceus-vrf/core/chains/evm/config/toml"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/config/docs"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/services/chainlink"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/services/chainlink/cfgtest"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/services/keystore/keys/ethkey"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/store/models"
)

func TestDoc(t *testing.T) {
	d := gotoml.NewDecoder(strings.NewReader(docs.DocsTOML))
	d.DisallowUnknownFields() // Ensure no extra fields
	var c chainlink.Config
	err := d.Decode(&c)
	var strict *gotoml.StrictMissingError
	if err != nil && strings.Contains(err.Error(), "undecoded keys: ") {
		t.Errorf("Docs contain extra fields: %v", err)
	} else if errors.As(err, &strict) {
		t.Fatal("StrictMissingError:", strict.String())
	} else {
		require.NoError(t, err)
	}

	// Except for TelemetryIngress.ServerPubKey and TelemetryIngress.URL as this will be removed in the future
	// and its only use is to signal to NOPs that these fields are no longer allowed
	emptyString := ""
	c.TelemetryIngress.ServerPubKey = &emptyString
	c.TelemetryIngress.URL = new(models.URL)

	cfgtest.AssertFieldsNotNil(t, c)

	var defaults chainlink.Config
	require.NoError(t, cfgtest.DocDefaultsOnly(strings.NewReader(docs.DocsTOML), &defaults, config.DecodeTOML))

	t.Run("EVM", func(t *testing.T) {
		fallbackDefaults := evmcfg.Defaults(nil)
		docDefaults := defaults.EVM[0].Chain

		require.Equal(t, "", *docDefaults.ChainType)
		docDefaults.ChainType = nil

		// clean up KeySpecific as a special case
		require.Equal(t, 1, len(docDefaults.KeySpecific))
		ks := evmcfg.KeySpecific{Key: new(ethkey.EIP55Address),
			GasEstimator: evmcfg.KeySpecificGasEstimator{PriceMax: new(assets.Wei)}}
		require.Equal(t, ks, docDefaults.KeySpecific[0])
		docDefaults.KeySpecific = nil

		// EVM.GasEstimator.BumpTxDepth doesn't have a constant default - it is derived from another field
		require.Zero(t, *docDefaults.GasEstimator.BumpTxDepth)
		docDefaults.GasEstimator.BumpTxDepth = nil

		require.Zero(t, *docDefaults.GasEstimator.LimitJobType.DR)
		require.Zero(t, *docDefaults.GasEstimator.LimitJobType.Keeper)
		require.Zero(t, *docDefaults.GasEstimator.LimitJobType.VRF)
		require.Zero(t, *docDefaults.GasEstimator.LimitJobType.FM)
		docDefaults.GasEstimator.LimitJobType = evmcfg.GasLimitJobType{}

		// EIP1559FeeCapBufferBlocks doesn't have a constant default - it is derived from another field
		require.Zero(t, *docDefaults.GasEstimator.BlockHistory.EIP1559FeeCapBufferBlocks)
		docDefaults.GasEstimator.BlockHistory.EIP1559FeeCapBufferBlocks = nil

		// addresses w/o global values
		require.Zero(t, *docDefaults.FlagsContractAddress)
		require.Zero(t, *docDefaults.LinkContractAddress)
		require.Zero(t, *docDefaults.OperatorFactoryAddress)
		docDefaults.FlagsContractAddress = nil
		docDefaults.LinkContractAddress = nil
		docDefaults.OperatorFactoryAddress = nil

		assertTOML(t, fallbackDefaults, docDefaults)
	})

}

func assertTOML[T any](t *testing.T, fallback, docs T) {
	t.Helper()
	t.Logf("fallback: %#v", fallback)
	t.Logf("docs: %#v", docs)
	fb, err := gotoml.Marshal(fallback)
	require.NoError(t, err)
	db, err := gotoml.Marshal(docs)
	require.NoError(t, err)
	fs, ds := string(fb), string(db)
	assert.Equal(t, fs, ds, diff.Diff(fs, ds))
}
