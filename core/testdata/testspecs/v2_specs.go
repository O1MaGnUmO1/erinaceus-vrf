package testspecs

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/smartcontractkit/chainlink/v2/core/chains/evm/assets"
	"github.com/smartcontractkit/chainlink/v2/core/chains/evm/utils"
	"github.com/smartcontractkit/chainlink/v2/core/services/vrf/vrfcommon"
)

type VRFSpecParams struct {
	JobID                         string
	Name                          string
	CoordinatorAddress            string
	VRFVersion                    vrfcommon.Version
	BatchCoordinatorAddress       string
	VRFOwnerAddress               string
	BatchFulfillmentEnabled       bool
	CustomRevertsPipelineEnabled  bool
	BatchFulfillmentGasMultiplier float64
	MinIncomingConfirmations      int
	FromAddresses                 []string
	PublicKey                     string
	ObservationSource             string
	EVMChainID                    string
	RequestedConfsDelay           int
	RequestTimeout                time.Duration
	V2                            bool
	ChunkSize                     int
	BackoffInitialDelay           time.Duration
	BackoffMaxDelay               time.Duration
	GasLanePrice                  *assets.Wei
	PollPeriod                    time.Duration
}

type VRFSpec struct {
	VRFSpecParams
	toml string
}

func (vs VRFSpec) Toml() string {
	return vs.toml
}

func GenerateVRFSpec(params VRFSpecParams) VRFSpec {
	jobID := "123e4567-e89b-12d3-a456-426655440000"
	if params.JobID != "" {
		jobID = params.JobID
	}
	name := "vrf-primary"
	if params.Name != "" {
		name = params.Name
	}
	vrfVersion := vrfcommon.V2
	if params.VRFVersion != "" {
		vrfVersion = params.VRFVersion
	}
	coordinatorAddress := "0xABA5eDc1a551E55b1A570c0e1f1055e5BE11eca7"
	if params.CoordinatorAddress != "" {
		coordinatorAddress = params.CoordinatorAddress
	}
	batchCoordinatorAddress := "0x5C7B1d96CA3132576A84423f624C2c492f668Fea"
	if params.BatchCoordinatorAddress != "" {
		batchCoordinatorAddress = params.BatchCoordinatorAddress
	}
	vrfOwnerAddress := "0x5383C25DA15b1253463626243215495a3718beE4"
	if params.VRFOwnerAddress != "" && vrfVersion == vrfcommon.V2 {
		vrfOwnerAddress = params.VRFOwnerAddress
	}
	fromAddesses := []string{"0x6f4BEe27f26cD4f8047bD70d2e74C190b43E807c"}
	if len(params.FromAddresses) != 0 {
		fromAddesses = params.FromAddresses
	}
	pollPeriod := 5 * time.Second
	if params.PollPeriod > 0 && vrfVersion == vrfcommon.V2 {
		pollPeriod = params.PollPeriod
	}
	batchFulfillmentGasMultiplier := 1.0
	if params.BatchFulfillmentGasMultiplier >= 1.0 {
		batchFulfillmentGasMultiplier = params.BatchFulfillmentGasMultiplier
	}
	confirmations := 6
	if params.MinIncomingConfirmations != 0 {
		confirmations = params.MinIncomingConfirmations
	}
	gasLanePrice := assets.GWei(100)
	if params.GasLanePrice != nil {
		gasLanePrice = params.GasLanePrice
	}
	requestTimeout := 24 * time.Hour
	if params.RequestTimeout != 0 {
		requestTimeout = params.RequestTimeout
	}
	publicKey := "0x79BE667EF9DCBBAC55A06295CE870B07029BFCDB2DCE28D959F2815B16F8179800"
	if params.PublicKey != "" {
		publicKey = params.PublicKey
	}
	chunkSize := 20
	if params.ChunkSize != 0 {
		chunkSize = params.ChunkSize
	}
	var observationSource string
	observationSource = fmt.Sprintf(`
decode_log   [type=ethabidecodelog
              abi="RandomWordsRequested(bytes32 indexed keyHash,uint256 requestId,uint256 preSeed,uint64 indexed subId,uint16 minimumRequestConfirmations,uint32 callbackGasLimit,uint32 numWords,address indexed sender)"
              data="$(jobRun.logData)"
              topics="$(jobRun.logTopics)"]
vrf          [type=vrfv2
              publicKey="$(jobSpec.publicKey)"
              requestBlockHash="$(jobRun.logBlockHash)"
              requestBlockNumber="$(jobRun.logBlockNumber)"
              topics="$(jobRun.logTopics)"]
estimate_gas [type=estimategaslimit
              to="%s"
              multiplier="1.1"
              data="$(vrf.output)"
]
simulate [type=ethcall
          to="%s"
		  gas="$(estimate_gas)"
		  gasPrice="$(jobSpec.maxGasPrice)"
		  extractRevertReason=true
		  contract="%s"
		  data="$(vrf.output)"
]
decode_log->vrf->estimate_gas->simulate
`, coordinatorAddress, coordinatorAddress, coordinatorAddress)

	if params.ObservationSource != "" {
		observationSource = params.ObservationSource
	}
	if params.EVMChainID == "" {
		params.EVMChainID = "0"
	}
	template := `
externalJobID = "%s"
type = "vrf"
schemaVersion = 1
name = "%s"
coordinatorAddress = "%s"
evmChainID         =  "%s"
batchCoordinatorAddress = "%s"
batchFulfillmentEnabled = %v
batchFulfillmentGasMultiplier = %s
customRevertsPipelineEnabled = %v
minIncomingConfirmations = %d
requestedConfsDelay = %d
requestTimeout = "%s"
publicKey = "%s"
chunkSize = %d
backoffInitialDelay = "%s"
backoffMaxDelay = "%s"
gasLanePrice = "%s"
pollPeriod = "%s"
observationSource = """
%s
"""
`
	toml := fmt.Sprintf(template,
		jobID, name, coordinatorAddress, params.EVMChainID, batchCoordinatorAddress,
		params.BatchFulfillmentEnabled, strconv.FormatFloat(batchFulfillmentGasMultiplier, 'f', 2, 64),
		params.CustomRevertsPipelineEnabled,
		confirmations, params.RequestedConfsDelay, requestTimeout.String(), publicKey, chunkSize,
		params.BackoffInitialDelay.String(), params.BackoffMaxDelay.String(), gasLanePrice.String(),
		pollPeriod.String(), observationSource)
	if len(params.FromAddresses) != 0 {
		var addresses []string
		for _, address := range params.FromAddresses {
			addresses = append(addresses, fmt.Sprintf("%q", address))
		}
		toml = toml + "\n" + fmt.Sprintf(`fromAddresses = [%s]`, strings.Join(addresses, ", "))
	} else {
		var addresses []string
		for _, address := range fromAddesses {
			addresses = append(addresses, fmt.Sprintf("%q", address))
		}
		toml = toml + "\n" + fmt.Sprintf(`fromAddresses = [%s]`, strings.Join(addresses, ", "))
	}
	if vrfVersion == vrfcommon.V2 {
		toml = toml + "\n" + fmt.Sprintf(`vrfOwnerAddress = "%s"`, vrfOwnerAddress)
	}

	return VRFSpec{VRFSpecParams: VRFSpecParams{
		JobID:                    jobID,
		Name:                     name,
		CoordinatorAddress:       coordinatorAddress,
		BatchCoordinatorAddress:  batchCoordinatorAddress,
		BatchFulfillmentEnabled:  params.BatchFulfillmentEnabled,
		MinIncomingConfirmations: confirmations,
		PublicKey:                publicKey,
		ObservationSource:        observationSource,
		EVMChainID:               params.EVMChainID,
		RequestedConfsDelay:      params.RequestedConfsDelay,
		RequestTimeout:           requestTimeout,
		ChunkSize:                chunkSize,
		BackoffInitialDelay:      params.BackoffInitialDelay,
		BackoffMaxDelay:          params.BackoffMaxDelay,
		VRFOwnerAddress:          vrfOwnerAddress,
		VRFVersion:               vrfVersion,
		PollPeriod:               pollPeriod,
	}, toml: toml}
}

// BlockhashStoreSpecParams defines params for building a blockhash store job spec.
type BlockhashStoreSpecParams struct {
	JobID                          string
	Name                           string
	CoordinatorV1Address           string
	CoordinatorV2Address           string
	CoordinatorV2PlusAddress       string
	WaitBlocks                     int
	HeartbeatPeriod                time.Duration
	LookbackBlocks                 int
	BlockhashStoreAddress          string
	TrustedBlockhashStoreAddress   string
	TrustedBlockhashStoreBatchSize int32
	PollPeriod                     time.Duration
	RunTimeout                     time.Duration
	EVMChainID                     int64
	FromAddresses                  []string
}

// BlockhashStoreSpec defines a blockhash store job spec.
type BlockhashStoreSpec struct {
	BlockhashStoreSpecParams
	toml string
}

// Toml returns the BlockhashStoreSpec in TOML string form.
func (bhs BlockhashStoreSpec) Toml() string {
	return bhs.toml
}

// GenerateBlockhashStoreSpec creates a BlockhashStoreSpec from the given params.
func GenerateBlockhashStoreSpec(params BlockhashStoreSpecParams) BlockhashStoreSpec {
	if params.JobID == "" {
		params.JobID = "123e4567-e89b-12d3-a456-426655442222"
	}

	if params.Name == "" {
		params.Name = "blockhash-store"
	}

	if params.CoordinatorV1Address == "" {
		params.CoordinatorV1Address = "0x19D20b4Ec0424A530C3C1cDe874445E37747eb18"
	}

	if params.CoordinatorV2Address == "" {
		params.CoordinatorV2Address = "0x2498e651Ae17C2d98417C4826F0816Ac6366A95E"
	}

	if params.CoordinatorV2PlusAddress == "" {
		params.CoordinatorV2PlusAddress = "0x92B5e28Ac583812874e4271380c7d070C5FB6E6b"
	}

	if params.TrustedBlockhashStoreAddress == "" {
		params.TrustedBlockhashStoreAddress = utils.ZeroAddress.Hex()
	}

	if params.TrustedBlockhashStoreBatchSize == 0 {
		params.TrustedBlockhashStoreBatchSize = 20
	}

	if params.WaitBlocks == 0 {
		params.WaitBlocks = 100
	}

	if params.LookbackBlocks == 0 {
		params.LookbackBlocks = 200
	}

	if params.BlockhashStoreAddress == "" {
		params.BlockhashStoreAddress = "0x31Ca8bf590360B3198749f852D5c516c642846F6"
	}

	if params.PollPeriod == 0 {
		params.PollPeriod = 30 * time.Second
	}

	if params.RunTimeout == 0 {
		params.RunTimeout = 15 * time.Second
	}

	var formattedFromAddresses string
	if params.FromAddresses == nil {
		formattedFromAddresses = `["0x4bd43cb108Bc3742e484f47E69EBfa378cb6278B"]`
	} else {
		var addresses []string
		for _, address := range params.FromAddresses {
			addresses = append(addresses, fmt.Sprintf("%q", address))
		}
		formattedFromAddresses = fmt.Sprintf("[%s]", strings.Join(addresses, ", "))
	}

	template := `
type = "blockhashstore"
schemaVersion = 1
name = "%s"
coordinatorV1Address = "%s"
coordinatorV2Address = "%s"
coordinatorV2PlusAddress = "%s"
waitBlocks = %d
lookbackBlocks = %d
blockhashStoreAddress = "%s"
trustedBlockhashStoreAddress = "%s"
trustedBlockhashStoreBatchSize = %d
pollPeriod = "%s"
runTimeout = "%s"
evmChainID = "%d"
fromAddresses = %s
heartbeatPeriod = "%s"
`
	toml := fmt.Sprintf(template, params.Name, params.CoordinatorV1Address,
		params.CoordinatorV2Address, params.CoordinatorV2PlusAddress, params.WaitBlocks, params.LookbackBlocks,
		params.BlockhashStoreAddress, params.TrustedBlockhashStoreAddress, params.TrustedBlockhashStoreBatchSize, params.PollPeriod.String(), params.RunTimeout.String(),
		params.EVMChainID, formattedFromAddresses, params.HeartbeatPeriod.String())

	return BlockhashStoreSpec{BlockhashStoreSpecParams: params, toml: toml}
}

// BlockHeaderFeederSpecParams defines params for building a block header feeder job spec.
type BlockHeaderFeederSpecParams struct {
	JobID                      string
	Name                       string
	CoordinatorV1Address       string
	CoordinatorV2Address       string
	CoordinatorV2PlusAddress   string
	WaitBlocks                 int
	LookbackBlocks             int
	BlockhashStoreAddress      string
	BatchBlockhashStoreAddress string
	PollPeriod                 time.Duration
	RunTimeout                 time.Duration
	EVMChainID                 int64
	FromAddresses              []string
	GetBlockhashesBatchSize    uint16
	StoreBlockhashesBatchSize  uint16
}

// BlockHeaderFeederSpec defines a block header feeder job spec.
type BlockHeaderFeederSpec struct {
	BlockHeaderFeederSpecParams
	toml string
}

// Toml returns the BlockhashStoreSpec in TOML string form.
func (b BlockHeaderFeederSpec) Toml() string {
	return b.toml
}

// GenerateBlockHeaderFeederSpec creates a BlockHeaderFeederSpec from the given params.
func GenerateBlockHeaderFeederSpec(params BlockHeaderFeederSpecParams) BlockHeaderFeederSpec {
	if params.JobID == "" {
		params.JobID = "123e4567-e89b-12d3-a456-426655442211"
	}

	if params.Name == "" {
		params.Name = "blockheaderfeeder"
	}

	if params.CoordinatorV1Address == "" {
		params.CoordinatorV1Address = "0x2d7F888fE0dD469bd81A12f77e6291508f714d4B"
	}

	if params.CoordinatorV2Address == "" {
		params.CoordinatorV2Address = "0x2d7F888fE0dD469bd81A12f77e6291508f714d4B"
	}

	if params.CoordinatorV2PlusAddress == "" {
		params.CoordinatorV2PlusAddress = "0x2d7F888fE0dD469bd81A12f77e6291508f714d4B"
	}

	if params.WaitBlocks == 0 {
		params.WaitBlocks = 256
	}

	if params.LookbackBlocks == 0 {
		params.LookbackBlocks = 500
	}

	if params.BlockhashStoreAddress == "" {
		params.BlockhashStoreAddress = "0x016D54091ee83D42aF46e4F2d7177D0A232D2bDa"
	}

	if params.BatchBlockhashStoreAddress == "" {
		params.BatchBlockhashStoreAddress = "0xde08B57586839BfF5DB58Bdd7FdeB7142Bff3795"
	}

	if params.PollPeriod == 0 {
		params.PollPeriod = 60 * time.Second
	}

	if params.RunTimeout == 0 {
		params.RunTimeout = 30 * time.Second
	}

	if params.GetBlockhashesBatchSize == 0 {
		params.GetBlockhashesBatchSize = 10
	}

	if params.StoreBlockhashesBatchSize == 0 {
		params.StoreBlockhashesBatchSize = 5
	}

	var formattedFromAddresses string
	if params.FromAddresses == nil {
		formattedFromAddresses = `["0xBe0b739f841bC113D4F4e4CdD16086ffAbB5f39f"]`
	} else {
		var addresses []string
		for _, address := range params.FromAddresses {
			addresses = append(addresses, fmt.Sprintf("%q", address))
		}
		formattedFromAddresses = fmt.Sprintf("[%s]", strings.Join(addresses, ", "))
	}

	template := `
type = "blockheaderfeeder"
schemaVersion = 1
name = "%s"
coordinatorV1Address = "%s"
coordinatorV2Address = "%s"
coordinatorV2PlusAddress = "%s"
waitBlocks = %d
lookbackBlocks = %d
blockhashStoreAddress = "%s"
batchBlockhashStoreAddress = "%s"
pollPeriod = "%s"
runTimeout = "%s"
evmChainID = "%d"
fromAddresses = %s
getBlockhashesBatchSize = %d
storeBlockhashesBatchSize = %d
`
	toml := fmt.Sprintf(template, params.Name, params.CoordinatorV1Address,
		params.CoordinatorV2Address, params.CoordinatorV2PlusAddress, params.WaitBlocks, params.LookbackBlocks,
		params.BlockhashStoreAddress, params.BatchBlockhashStoreAddress, params.PollPeriod.String(),
		params.RunTimeout.String(), params.EVMChainID, formattedFromAddresses, params.GetBlockhashesBatchSize,
		params.StoreBlockhashesBatchSize)

	return BlockHeaderFeederSpec{BlockHeaderFeederSpecParams: params, toml: toml}
}
