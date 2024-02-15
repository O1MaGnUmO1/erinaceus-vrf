package vrftesthelpers

import (
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink/v2/core/chains/evm/assets"
	"github.com/smartcontractkit/chainlink/v2/core/gethwrappers/generated/link_token_interface"
	"github.com/smartcontractkit/chainlink/v2/core/gethwrappers/generated/solidity_vrf_consumer_interface"
	"github.com/smartcontractkit/chainlink/v2/core/gethwrappers/generated/solidity_vrf_consumer_interface_v08"
	"github.com/smartcontractkit/chainlink/v2/core/gethwrappers/generated/solidity_vrf_coordinator_interface"
	"github.com/smartcontractkit/chainlink/v2/core/gethwrappers/generated/solidity_vrf_request_id"
	"github.com/smartcontractkit/chainlink/v2/core/gethwrappers/generated/solidity_vrf_request_id_v08"
	"github.com/smartcontractkit/chainlink/v2/core/internal/cltest"
	"github.com/smartcontractkit/chainlink/v2/core/internal/testutils"
	"github.com/smartcontractkit/chainlink/v2/core/services/keystore/keys/ethkey"
	"github.com/smartcontractkit/chainlink/v2/core/services/keystore/keys/vrfkey"
	"github.com/smartcontractkit/chainlink/v2/core/services/vrf/proof"
)

var (
	WeiPerUnitLink = decimal.RequireFromString("10000000000000000")
)

func GenerateProofResponseFromProof(p vrfkey.Proof, s proof.PreSeedData) (
	proof.MarshaledOnChainResponse, error) {
	return proof.GenerateProofResponseFromProof(p, s)
}

// CoordinatorUniverse represents the universe in which a randomness request occurs and
// is fulfilled.
type CoordinatorUniverse struct {
	// Golang wrappers ofr solidity contracts
	RootContract               *solidity_vrf_coordinator_interface.VRFCoordinator
	LinkContract               *link_token_interface.LinkToken
	ConsumerContract           *solidity_vrf_consumer_interface.VRFConsumer
	RequestIDBase              *solidity_vrf_request_id.VRFRequestIDBaseTestHelper
	ConsumerContractV08        *solidity_vrf_consumer_interface_v08.VRFConsumer
	RequestIDBaseV08           *solidity_vrf_request_id_v08.VRFRequestIDBaseTestHelper
	RootContractAddress        common.Address
	ConsumerContractAddress    common.Address
	ConsumerContractAddressV08 common.Address
	LinkContractAddress        common.Address
	BHSContractAddress         common.Address

	// Abstraction representation of the ethereum blockchain
	Backend        *backends.SimulatedBackend
	CoordinatorABI *abi.ABI
	ConsumerABI    *abi.ABI
	// Cast of participants
	Sergey *bind.TransactOpts // Owns all the LINK initially
	Neil   *bind.TransactOpts // Node operator running VRF service
	Ned    *bind.TransactOpts // Secondary node operator
	Carol  *bind.TransactOpts // Author of consuming contract which requests randomness
}

var oneEth = big.NewInt(1000000000000000000) // 1e18 wei

func NewVRFCoordinatorUniverseWithV08Consumer(t *testing.T, key ethkey.KeyV2) CoordinatorUniverse {
	cu := NewVRFCoordinatorUniverse(t, key)
	consumerContractAddress, _, consumerContract, err :=
		solidity_vrf_consumer_interface_v08.DeployVRFConsumer(
			cu.Carol, cu.Backend, cu.RootContractAddress, cu.LinkContractAddress)
	require.NoError(t, err, "failed to deploy v08 VRFConsumer contract to simulated ethereum blockchain")
	_, _, requestIDBase, err :=
		solidity_vrf_request_id_v08.DeployVRFRequestIDBaseTestHelper(cu.Neil, cu.Backend)
	require.NoError(t, err, "failed to deploy v08 VRFRequestIDBaseTestHelper contract to simulated ethereum blockchain")
	cu.ConsumerContractAddressV08 = consumerContractAddress
	cu.RequestIDBaseV08 = requestIDBase
	cu.ConsumerContractV08 = consumerContract
	_, err = cu.LinkContract.Transfer(cu.Sergey, consumerContractAddress, oneEth) // Actually, LINK
	require.NoError(t, err, "failed to send LINK to VRFConsumer contract on simulated ethereum blockchain")
	cu.Backend.Commit()
	return cu
}

// newVRFCoordinatorUniverse sets up all identities and contracts associated with
// testing the solidity VRF contracts involved in randomness request workflow
func NewVRFCoordinatorUniverse(t *testing.T, keys ...ethkey.KeyV2) CoordinatorUniverse {
	var oracleTransactors []*bind.TransactOpts
	for _, key := range keys {
		oracleTransactor, err := bind.NewKeyedTransactorWithChainID(key.ToEcdsaPrivKey(), testutils.SimulatedChainID)
		require.NoError(t, err)
		oracleTransactors = append(oracleTransactors, oracleTransactor)
	}

	var (
		sergey = testutils.MustNewSimTransactor(t)
		neil   = testutils.MustNewSimTransactor(t)
		ned    = testutils.MustNewSimTransactor(t)
		carol  = testutils.MustNewSimTransactor(t)
	)
	genesisData := core.GenesisAlloc{
		sergey.From: {Balance: assets.Ether(1000).ToInt()},
		neil.From:   {Balance: assets.Ether(1000).ToInt()},
		ned.From:    {Balance: assets.Ether(1000).ToInt()},
		carol.From:  {Balance: assets.Ether(1000).ToInt()},
	}

	for _, t := range oracleTransactors {
		genesisData[t.From] = core.GenesisAccount{Balance: assets.Ether(1000).ToInt()}
	}

	gasLimit := uint32(ethconfig.Defaults.Miner.GasCeil)
	consumerABI, err := abi.JSON(strings.NewReader(
		solidity_vrf_consumer_interface.VRFConsumerABI))
	require.NoError(t, err)
	coordinatorABI, err := abi.JSON(strings.NewReader(
		solidity_vrf_coordinator_interface.VRFCoordinatorABI))
	require.NoError(t, err)
	backend := cltest.NewSimulatedBackend(t, genesisData, gasLimit)
	linkAddress, _, linkContract, err := link_token_interface.DeployLinkToken(
		sergey, backend)
	require.NoError(t, err, "failed to deploy link contract to simulated ethereum blockchain")
	coordinatorAddress, _, coordinatorContract, err :=
		solidity_vrf_coordinator_interface.DeployVRFCoordinator(
			neil, backend, linkAddress, common.Address{})
	require.NoError(t, err, "failed to deploy VRFCoordinator contract to simulated ethereum blockchain")
	consumerContractAddress, _, consumerContract, err :=
		solidity_vrf_consumer_interface.DeployVRFConsumer(
			carol, backend, coordinatorAddress, linkAddress)
	require.NoError(t, err, "failed to deploy VRFConsumer contract to simulated ethereum blockchain")
	_, _, requestIDBase, err :=
		solidity_vrf_request_id.DeployVRFRequestIDBaseTestHelper(neil, backend)
	require.NoError(t, err, "failed to deploy VRFRequestIDBaseTestHelper contract to simulated ethereum blockchain")
	_, err = linkContract.Transfer(sergey, consumerContractAddress, oneEth) // Actually, LINK
	require.NoError(t, err, "failed to send LINK to VRFConsumer contract on simulated ethereum blockchain")
	backend.Commit()
	return CoordinatorUniverse{
		RootContract:            coordinatorContract,
		RootContractAddress:     coordinatorAddress,
		LinkContract:            linkContract,
		LinkContractAddress:     linkAddress,
		ConsumerContract:        consumerContract,
		RequestIDBase:           requestIDBase,
		ConsumerContractAddress: consumerContractAddress,
		Backend:                 backend,
		CoordinatorABI:          &coordinatorABI,
		ConsumerABI:             &consumerABI,
		Sergey:                  sergey,
		Neil:                    neil,
		Ned:                     ned,
		Carol:                   carol,
	}
}
