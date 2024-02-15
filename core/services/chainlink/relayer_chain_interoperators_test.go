package chainlink_test

import (
	"errors"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/O1MaGnUmO1/chainlink-common/pkg/loop"
	"github.com/O1MaGnUmO1/chainlink-common/pkg/utils/mailbox"
	ubig "github.com/smartcontractkit/chainlink/v2/core/chains/evm/utils/big"
	"github.com/smartcontractkit/chainlink/v2/core/chains/legacyevm"
	"github.com/smartcontractkit/chainlink/v2/core/internal/cltest"
	"github.com/smartcontractkit/chainlink/v2/core/internal/testutils"
	"github.com/smartcontractkit/chainlink/v2/core/internal/testutils/configtest"
	"github.com/smartcontractkit/chainlink/v2/core/internal/testutils/pgtest"
	"github.com/smartcontractkit/chainlink/v2/core/logger"
	"github.com/smartcontractkit/chainlink/v2/core/services/chainlink"
	"github.com/smartcontractkit/chainlink/v2/core/services/pg"
	"github.com/smartcontractkit/chainlink/v2/core/services/relay"
	"github.com/smartcontractkit/chainlink/v2/core/store/models"
	"github.com/smartcontractkit/chainlink/v2/plugins"

	evmcfg "github.com/smartcontractkit/chainlink/v2/core/chains/evm/config/toml"
)

func TestCoreRelayerChainInteroperators(t *testing.T) {

	evmChainID1, evmChainID2 := ubig.New(big.NewInt(1)), ubig.New(big.NewInt(2))

	cfg := configtest.NewGeneralConfig(t, func(c *chainlink.Config, s *chainlink.Secrets) {

		cfg := evmcfg.Defaults(evmChainID1)
		node1_1 := evmcfg.Node{
			Name:     ptr("Test node chain1:1"),
			WSURL:    models.MustParseURL("ws://localhost:8546"),
			HTTPURL:  models.MustParseURL("http://localhost:8546"),
			SendOnly: ptr(false),
			Order:    ptr(int32(15)),
		}
		node1_2 := evmcfg.Node{
			Name:     ptr("Test node chain1:2"),
			WSURL:    models.MustParseURL("ws://localhost:8547"),
			HTTPURL:  models.MustParseURL("http://localhost:8547"),
			SendOnly: ptr(false),
			Order:    ptr(int32(36)),
		}
		node2_1 := evmcfg.Node{
			Name:     ptr("Test node chain2:1"),
			WSURL:    models.MustParseURL("ws://localhost:8547"),
			HTTPURL:  models.MustParseURL("http://localhost:8547"),
			SendOnly: ptr(false),
			Order:    ptr(int32(11)),
		}
		c.EVM[0] = &evmcfg.EVMConfig{
			ChainID: evmChainID1,
			Enabled: ptr(true),
			Chain:   cfg,
			Nodes:   evmcfg.EVMNodes{&node1_1, &node1_2},
		}
		id2 := ubig.New(big.NewInt(2))
		c.EVM = append(c.EVM, &evmcfg.EVMConfig{
			ChainID: evmChainID2,
			Chain:   evmcfg.Defaults(id2),
			Enabled: ptr(true),
			Nodes:   evmcfg.EVMNodes{&node2_1},
		})
	})

	db := pgtest.NewSqlxDB(t)
	keyStore := cltest.NewKeyStore(t, db, cfg.Database())

	lggr := logger.TestLogger(t)

	factory := chainlink.RelayerFactory{
		Logger:       lggr,
		LoopRegistry: plugins.NewLoopRegistry(lggr, nil),
		GRPCOpts:     loop.GRPCOpts{},
	}

	testctx := testutils.Context(t)

	tests := []struct {
		name                    string
		initFuncs               []chainlink.CoreRelayerChainInitFunc
		expectedRelayerNetworks map[relay.Network]struct{}

		expectedEVMChainCnt   int
		expectedEVMNodeCnt    int
		expectedEVMRelayerIds []relay.ID

		expectedSolanaChainCnt   int
		expectedSolanaNodeCnt    int
		expectedSolanaRelayerIds []relay.ID

		expectedStarknetChainCnt   int
		expectedStarknetNodeCnt    int
		expectedStarknetRelayerIds []relay.ID

		expectedCosmosChainCnt   int
		expectedCosmosNodeCnt    int
		expectedCosmosRelayerIds []relay.ID
	}{

		{name: "2 evm chains with 3 nodes",
			initFuncs: []chainlink.CoreRelayerChainInitFunc{
				chainlink.InitEVM(testctx, factory, chainlink.EVMFactoryConfig{
					ChainOpts: legacyevm.ChainOpts{
						AppConfig:        cfg,
						EventBroadcaster: pg.NewNullEventBroadcaster(),
						MailMon:          &mailbox.Monitor{},
						DB:               db,
					},
					CSAETHKeystore: keyStore,
				}),
			},
			expectedEVMChainCnt: 2,
			expectedEVMNodeCnt:  3,
			expectedEVMRelayerIds: []relay.ID{
				{Network: relay.EVM, ChainID: relay.ChainID(evmChainID1.String())},
				{Network: relay.EVM, ChainID: relay.ChainID(evmChainID2.String())},
			},
			expectedRelayerNetworks: map[relay.Network]struct{}{relay.EVM: {}},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var cr *chainlink.CoreRelayerChainInteroperators
			{
				var err error
				cr, err = chainlink.NewCoreRelayerChainInteroperators(tt.initFuncs...)
				require.NoError(t, err)

				expectedChainCnt := tt.expectedEVMChainCnt + tt.expectedCosmosChainCnt + tt.expectedSolanaChainCnt + tt.expectedStarknetChainCnt
				allChainsStats, cnt, err := cr.ChainStatuses(testctx, 0, 0)
				assert.NoError(t, err)
				assert.Len(t, allChainsStats, expectedChainCnt)
				assert.Equal(t, cnt, len(allChainsStats))
				assert.Len(t, cr.Slice(), expectedChainCnt)

				// should be one relayer per chain and one service per relayer
				assert.Len(t, cr.Slice(), expectedChainCnt)
				assert.Len(t, cr.Services(), expectedChainCnt)

				expectedNodeCnt := tt.expectedEVMNodeCnt + tt.expectedCosmosNodeCnt + tt.expectedSolanaNodeCnt + tt.expectedStarknetNodeCnt
				allNodeStats, cnt, err := cr.NodeStatuses(testctx, 0, 0)
				assert.NoError(t, err)
				assert.Len(t, allNodeStats, expectedNodeCnt)
				assert.Equal(t, cnt, len(allNodeStats))
			}

			gotRelayerNetworks := make(map[relay.Network]struct{})
			for relayNetwork := range relay.SupportedRelays {
				var expectedChainCnt, expectedNodeCnt int
				switch relayNetwork {
				case relay.EVM:
					expectedChainCnt, expectedNodeCnt = tt.expectedEVMChainCnt, tt.expectedEVMNodeCnt
				case relay.Cosmos:
					expectedChainCnt, expectedNodeCnt = tt.expectedCosmosChainCnt, tt.expectedCosmosNodeCnt
				case relay.Solana:
					expectedChainCnt, expectedNodeCnt = tt.expectedSolanaChainCnt, tt.expectedSolanaNodeCnt
				case relay.StarkNet:
					expectedChainCnt, expectedNodeCnt = tt.expectedStarknetChainCnt, tt.expectedStarknetNodeCnt
				default:
					require.Fail(t, "untested relay network", relayNetwork)
				}

				interops := cr.List(chainlink.FilterRelayersByType(relayNetwork))
				assert.Len(t, cr.List(chainlink.FilterRelayersByType(relayNetwork)).Slice(), expectedChainCnt)
				if len(interops.Slice()) > 0 {
					gotRelayerNetworks[relayNetwork] = struct{}{}
				}

				// check legacy chains for those that haven't migrated fully to the loop relayer interface
				if relayNetwork == relay.EVM {
					_, wantEVM := tt.expectedRelayerNetworks[relay.EVM]
					if wantEVM {
						assert.Len(t, cr.LegacyEVMChains().Slice(), expectedChainCnt)
					} else {
						assert.Nil(t, cr.LegacyEVMChains())
					}
				}

				nodesStats, cnt, err := interops.NodeStatuses(testctx, 0, 0)
				assert.NoError(t, err)
				assert.Len(t, nodesStats, expectedNodeCnt)
				assert.Equal(t, cnt, len(nodesStats))

			}
			assert.EqualValues(t, gotRelayerNetworks, tt.expectedRelayerNetworks)

			allRelayerIds := [][]relay.ID{
				tt.expectedEVMRelayerIds,
				tt.expectedCosmosRelayerIds,
				tt.expectedSolanaRelayerIds,
				tt.expectedStarknetRelayerIds,
			}

			for _, chainSpecificRelayerIds := range allRelayerIds {
				for _, wantId := range chainSpecificRelayerIds {
					lr, err := cr.Get(wantId)
					assert.NotNil(t, lr)
					assert.NoError(t, err)
					stat, err := cr.ChainStatus(testctx, wantId)
					assert.NoError(t, err)
					assert.Equal(t, wantId.ChainID, stat.ID)
					// check legacy chains for evm and cosmos
					if wantId.Network == relay.EVM {
						c, err := cr.LegacyEVMChains().Get(wantId.ChainID)
						assert.NoError(t, err)
						assert.NotNil(t, c)
						assert.Equal(t, wantId.ChainID, c.ID().String())
					}
				}
			}

			expectedMissing := relay.ID{Network: relay.Cosmos, ChainID: "not a chain id"}
			unwanted, err := cr.Get(expectedMissing)
			assert.Nil(t, unwanted)
			assert.ErrorIs(t, err, chainlink.ErrNoSuchRelayer)

		})

	}

	t.Run("bad init func", func(t *testing.T) {
		t.Parallel()
		errBadFunc := errors.New("this is a bad func")
		badFunc := func() chainlink.CoreRelayerChainInitFunc {
			return func(op *chainlink.CoreRelayerChainInteroperators) error {
				return errBadFunc
			}
		}
		cr, err := chainlink.NewCoreRelayerChainInteroperators(badFunc())
		assert.Nil(t, cr)
		assert.ErrorIs(t, err, errBadFunc)
	})
}

func ptr[T any](t T) *T { return &t }
