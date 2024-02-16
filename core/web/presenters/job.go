package presenters

import (
	"time"

	"github.com/google/uuid"

	commonassets "github.com/O1MaGnUmO1/chainlink-common/pkg/assets"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/chains/evm/assets"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/chains/evm/utils/big"
	clnull "github.com/O1MaGnUmO1/erinaceus-vrf/core/null"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/services/job"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/services/keystore/keys/ethkey"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/services/pipeline"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/services/signatures/secp256k1"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/store/models"
)

// JobSpecType defines the the the spec type of the job
type JobSpecType string

func (t JobSpecType) String() string {
	return string(t)
}

const (
	DirectRequestJobSpec     JobSpecType = "directrequest"
	FluxMonitorJobSpec       JobSpecType = "fluxmonitor"
	OffChainReportingJobSpec JobSpecType = "offchainreporting"
	KeeperJobSpec            JobSpecType = "keeper"
	CronJobSpec              JobSpecType = "cron"
	VRFJobSpec               JobSpecType = "vrf"
	WebhookJobSpec           JobSpecType = "webhook"
	BlockhashStoreJobSpec    JobSpecType = "blockhashstore"
	BlockHeaderFeederJobSpec JobSpecType = "blockheaderfeeder"
	BootstrapJobSpec         JobSpecType = "bootstrap"
	GatewayJobSpec           JobSpecType = "gateway"
)

// DirectRequestSpec defines the spec details of a DirectRequest Job
type DirectRequestSpec struct {
	ContractAddress          ethkey.EIP55Address      `json:"contractAddress"`
	MinIncomingConfirmations clnull.Uint32            `json:"minIncomingConfirmations"`
	MinContractPayment       *commonassets.Link       `json:"minContractPaymentLinkJuels"`
	Requesters               models.AddressCollection `json:"requesters"`
	Initiator                string                   `json:"initiator"`
	CreatedAt                time.Time                `json:"createdAt"`
	UpdatedAt                time.Time                `json:"updatedAt"`
	EVMChainID               *big.Big                 `json:"evmChainID"`
}

// PipelineSpec defines the spec details of the pipeline
type PipelineSpec struct {
	ID           int32  `json:"id"`
	JobID        int32  `json:"jobID"`
	DotDAGSource string `json:"dotDagSource"`
}

// NewPipelineSpec generates a new PipelineSpec from a pipeline.Spec
func NewPipelineSpec(spec *pipeline.Spec) PipelineSpec {
	return PipelineSpec{
		ID:           spec.ID,
		JobID:        spec.JobID,
		DotDAGSource: spec.DotDagSource,
	}
}

// WebhookSpec defines the spec details of a Webhook Job
type WebhookSpec struct {
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// NewWebhookSpec generates a new WebhookSpec from a job.WebhookSpec
func NewWebhookSpec(spec *job.WebhookSpec) *WebhookSpec {
	return &WebhookSpec{
		CreatedAt: spec.CreatedAt,
		UpdatedAt: spec.UpdatedAt,
	}
}

type VRFSpec struct {
	BatchCoordinatorAddress       *ethkey.EIP55Address  `json:"batchCoordinatorAddress"`
	BatchFulfillmentEnabled       bool                  `json:"batchFulfillmentEnabled"`
	CustomRevertsPipelineEnabled  *bool                 `json:"customRevertsPipelineEnabled,omitempty"`
	BatchFulfillmentGasMultiplier float64               `json:"batchFulfillmentGasMultiplier"`
	CoordinatorAddress            ethkey.EIP55Address   `json:"coordinatorAddress"`
	PublicKey                     secp256k1.PublicKey   `json:"publicKey"`
	FromAddresses                 []ethkey.EIP55Address `json:"fromAddresses"`
	PollPeriod                    models.Duration       `json:"pollPeriod"`
	MinIncomingConfirmations      uint32                `json:"confirmations"`
	CreatedAt                     time.Time             `json:"createdAt"`
	UpdatedAt                     time.Time             `json:"updatedAt"`
	EVMChainID                    *big.Big              `json:"evmChainID"`
	ChunkSize                     uint32                `json:"chunkSize"`
	RequestTimeout                models.Duration       `json:"requestTimeout"`
	BackoffInitialDelay           models.Duration       `json:"backoffInitialDelay"`
	BackoffMaxDelay               models.Duration       `json:"backoffMaxDelay"`
	GasLanePrice                  *assets.Wei           `json:"gasLanePrice"`
	RequestedConfsDelay           int64                 `json:"requestedConfsDelay"`
	VRFOwnerAddress               *ethkey.EIP55Address  `json:"vrfOwnerAddress,omitempty"`
}

func NewVRFSpec(spec *job.VRFSpec) *VRFSpec {
	return &VRFSpec{
		BatchCoordinatorAddress:       spec.BatchCoordinatorAddress,
		BatchFulfillmentEnabled:       spec.BatchFulfillmentEnabled,
		BatchFulfillmentGasMultiplier: float64(spec.BatchFulfillmentGasMultiplier),
		CustomRevertsPipelineEnabled:  &spec.CustomRevertsPipelineEnabled,
		CoordinatorAddress:            spec.CoordinatorAddress,
		PublicKey:                     spec.PublicKey,
		FromAddresses:                 spec.FromAddresses,
		PollPeriod:                    models.MustMakeDuration(spec.PollPeriod),
		MinIncomingConfirmations:      spec.MinIncomingConfirmations,
		CreatedAt:                     spec.CreatedAt,
		UpdatedAt:                     spec.UpdatedAt,
		EVMChainID:                    spec.EVMChainID,
		ChunkSize:                     spec.ChunkSize,
		RequestTimeout:                models.MustMakeDuration(spec.RequestTimeout),
		BackoffInitialDelay:           models.MustMakeDuration(spec.BackoffInitialDelay),
		BackoffMaxDelay:               models.MustMakeDuration(spec.BackoffMaxDelay),
		GasLanePrice:                  spec.GasLanePrice,
		RequestedConfsDelay:           spec.RequestedConfsDelay,
		VRFOwnerAddress:               spec.VRFOwnerAddress,
	}
}

type GatewaySpec struct {
	GatewayConfig map[string]interface{} `json:"gatewayConfig"`
	CreatedAt     time.Time              `json:"createdAt"`
	UpdatedAt     time.Time              `json:"updatedAt"`
}

func NewGatewaySpec(spec *job.GatewaySpec) *GatewaySpec {
	return &GatewaySpec{
		GatewayConfig: spec.GatewayConfig,
		CreatedAt:     spec.CreatedAt,
		UpdatedAt:     spec.UpdatedAt,
	}
}

// JobError represents errors on the job
type JobError struct {
	ID          int64     `json:"id"`
	Description string    `json:"description"`
	Occurrences uint      `json:"occurrences"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func NewJobError(e job.SpecError) JobError {
	return JobError{
		ID:          e.ID,
		Description: e.Description,
		Occurrences: e.Occurrences,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}
}

// JobResource represents a JobResource
type JobResource struct {
	JAID
	Name              string          `json:"name"`
	Type              JobSpecType     `json:"type"`
	SchemaVersion     uint32          `json:"schemaVersion"`
	GasLimit          clnull.Uint32   `json:"gasLimit"`
	ForwardingAllowed bool            `json:"forwardingAllowed"`
	MaxTaskDuration   models.Interval `json:"maxTaskDuration"`
	ExternalJobID     uuid.UUID       `json:"externalJobID"`
	VRFSpec           *VRFSpec        `json:"vrfSpec"`
	WebhookSpec       *WebhookSpec    `json:"webhookSpec"`
	GatewaySpec       *GatewaySpec    `json:"gatewaySpec"`
	PipelineSpec      PipelineSpec    `json:"pipelineSpec"`
	Errors            []JobError      `json:"errors"`
}

// NewJobResource initializes a new JSONAPI job resource
func NewJobResource(j job.Job) *JobResource {
	resource := &JobResource{
		JAID:              NewJAIDInt32(j.ID),
		Name:              j.Name.ValueOrZero(),
		Type:              JobSpecType(j.Type),
		SchemaVersion:     j.SchemaVersion,
		GasLimit:          j.GasLimit,
		ForwardingAllowed: j.ForwardingAllowed,
		MaxTaskDuration:   j.MaxTaskDuration,
		PipelineSpec:      NewPipelineSpec(j.PipelineSpec),
		ExternalJobID:     j.ExternalJobID,
	}

	switch j.Type {
	case job.VRF:
		resource.VRFSpec = NewVRFSpec(j.VRFSpec)
	case job.Webhook:
		resource.WebhookSpec = NewWebhookSpec(j.WebhookSpec)
	case job.Gateway:
		resource.GatewaySpec = NewGatewaySpec(j.GatewaySpec)
	case job.LegacyGasStationServer, job.LegacyGasStationSidecar:
		// unsupported
	}

	jes := []JobError{}
	for _, e := range j.JobSpecErrors {
		jes = append(jes, NewJobError((e)))
	}
	resource.Errors = jes

	return resource
}

// GetName implements the api2go EntityNamer interface
func (r JobResource) GetName() string {
	return "jobs"
}
