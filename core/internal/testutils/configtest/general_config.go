package configtest

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/O1MaGnUmO1/erinaceus-vrf/core/chains/evm/client"
	evmclient "github.com/O1MaGnUmO1/erinaceus-vrf/core/chains/evm/client"
	evmcfg "github.com/O1MaGnUmO1/erinaceus-vrf/core/chains/evm/config/toml"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/chains/evm/utils/big"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/internal/testutils"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/services/erinaceus"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/store/dialects"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/store/models"
)

const DefaultPeerID = "12D3KooWPjceQrSwdWXPyLLeABRXmuqt69Rg3sBYbU1Nft9HyQ6X"

// NewTestGeneralConfig returns a new erinaceus.GeneralConfig with default test overrides and one chain with evmclient.NullClientChainID.
func NewTestGeneralConfig(t testing.TB) erinaceus.GeneralConfig { return NewGeneralConfig(t, nil) }

// NewGeneralConfig returns a new erinaceus.GeneralConfig with overrides.
// The default test overrides are applied before overrideFn, and include one chain with evmclient.NullClientChainID.
func NewGeneralConfig(t testing.TB, overrideFn func(*erinaceus.Config, *erinaceus.Secrets)) erinaceus.GeneralConfig {
	tempDir := t.TempDir()
	g, err := erinaceus.GeneralConfigOpts{
		OverrideFn: func(c *erinaceus.Config, s *erinaceus.Secrets) {
			overrides(c, s)
			c.RootDir = &tempDir
			if fn := overrideFn; fn != nil {
				fn(c, s)
			}
		},
	}.New()
	require.NoError(t, err)
	return g
}

// overrides applies some test config settings and adds a default chain with evmclient.NullClientChainID.
func overrides(c *erinaceus.Config, s *erinaceus.Secrets) {
	s.Password.Keystore = models.NewSecret("dummy-to-pass-validation")

	c.InsecureFastScrypt = ptr(true)
	c.ShutdownGracePeriod = models.MustNewDuration(testutils.DefaultWaitTimeout)

	c.Database.Dialect = dialects.TransactionWrappedPostgres
	c.Database.Lock.Enabled = ptr(false)
	c.Database.MaxIdleConns = ptr[int64](20)
	c.Database.MaxOpenConns = ptr[int64](20)
	c.Database.MigrateOnStartup = ptr(false)
	c.Database.DefaultLockTimeout = models.MustNewDuration(1 * time.Minute)

	c.JobPipeline.ReaperInterval = models.MustNewDuration(0)

	c.WebServer.SessionTimeout = models.MustNewDuration(2 * time.Minute)
	c.WebServer.BridgeResponseURL = models.MustParseURL("http://localhost:6688")
	testIP := net.ParseIP("127.0.0.1")
	c.WebServer.ListenIP = &testIP
	c.WebServer.TLS.ListenIP = &testIP

	chainID := big.NewI(evmclient.NullClientChainID)
	c.EVM = append(c.EVM, &evmcfg.EVMConfig{
		ChainID: chainID,
		Chain:   evmcfg.Defaults(chainID),
		Nodes: evmcfg.EVMNodes{
			&evmcfg.Node{
				Name:     ptr("test"),
				WSURL:    &models.URL{},
				HTTPURL:  &models.URL{},
				SendOnly: new(bool),
				Order:    ptr[int32](100),
			},
		},
	})
}

// NewGeneralConfigSimulated returns a new erinaceus.GeneralConfig with overrides, including the simulated EVM chain.
// The default test overrides are applied before overrideFn.
// The simulated chain (testutils.SimulatedChainID) replaces the null chain (evmclient.NullClientChainID).
func NewGeneralConfigSimulated(t testing.TB, overrideFn func(*erinaceus.Config, *erinaceus.Secrets)) erinaceus.GeneralConfig {
	return NewGeneralConfig(t, func(c *erinaceus.Config, s *erinaceus.Secrets) {
		simulated(c, s)
		if fn := overrideFn; fn != nil {
			fn(c, s)
		}
	})
}

// simulated is a config override func that appends the simulated EVM chain (testutils.SimulatedChainID),
// or replaces the null chain (client.NullClientChainID) if that is the only entry.
func simulated(c *erinaceus.Config, s *erinaceus.Secrets) {
	chainID := big.New(testutils.SimulatedChainID)
	enabled := true
	cfg := evmcfg.EVMConfig{
		ChainID: chainID,
		Chain:   evmcfg.Defaults(chainID),
		Enabled: &enabled,
		Nodes:   evmcfg.EVMNodes{&validTestNode},
	}
	if len(c.EVM) == 1 && c.EVM[0].ChainID.Cmp(big.NewI(client.NullClientChainID)) == 0 {
		c.EVM[0] = &cfg // replace null, if only entry
	} else {
		c.EVM = append(c.EVM, &cfg)
	}
}

var validTestNode = evmcfg.Node{
	Name:     ptr("simulated-node"),
	WSURL:    models.MustParseURL("WSS://simulated-wss.com/ws"),
	HTTPURL:  models.MustParseURL("http://simulated.com"),
	SendOnly: nil,
	Order:    ptr(int32(1)),
}

func ptr[T any](v T) *T { return &v }
