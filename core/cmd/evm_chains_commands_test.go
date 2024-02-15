package cmd_test

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	client2 "github.com/O1MaGnUmO1/erinaceus-vrf/core/chains/evm/client"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/chains/evm/utils/big"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/cmd"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/internal/cltest"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/internal/testutils"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/services/erinaceus"
)

func newRandChainID() *big.Big {
	return big.New(testutils.NewRandomEVMChainID())
}

func TestShell_IndexEVMChains(t *testing.T) {
	t.Parallel()

	app := startNewApplicationV2(t, func(c *erinaceus.Config, s *erinaceus.Secrets) {
		c.EVM[0].Enabled = ptr(true)
		c.EVM[0].NonceAutoSync = ptr(false)
		c.EVM[0].BalanceMonitor.Enabled = ptr(false)
	})
	client, r := app.NewShellAndRenderer()

	require.Nil(t, cmd.EVMChainClient(client).IndexChains(cltest.EmptyCLIContext()))
	chains := *r.Renders[0].(*cmd.EVMChainPresenters)
	require.Len(t, chains, 1)
	c := chains[0]
	assert.Equal(t, strconv.Itoa(client2.NullClientChainID), c.ID)
	assertTableRenders(t, r)
}
