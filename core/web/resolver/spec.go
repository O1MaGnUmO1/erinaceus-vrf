package resolver

import (
	"github.com/graph-gophers/graphql-go"

	"github.com/O1MaGnUmO1/erinaceus-vrf/core/services/job"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/utils/stringutils"
	"github.com/O1MaGnUmO1/erinaceus-vrf/core/web/gqlscalar"
)

type SpecResolver struct {
	j job.Job
}

func NewSpec(j job.Job) *SpecResolver {
	return &SpecResolver{j: j}
}

func (r *SpecResolver) ToVRFSpec() (*VRFSpecResolver, bool) {
	if r.j.Type != job.VRF {
		return nil, false
	}

	return &VRFSpecResolver{spec: *r.j.VRFSpec}, true
}

func (r *SpecResolver) ToWebhookSpec() (*WebhookSpecResolver, bool) {
	if r.j.Type != job.Webhook {
		return nil, false
	}

	return &WebhookSpecResolver{spec: *r.j.WebhookSpec}, true
}

// ToBlockhashStoreSpec returns the BlockhashStoreSpec from the SpecResolver if the job is a
// BlockhashStore job.
func (r *SpecResolver) ToBlockhashStoreSpec() (*BlockhashStoreSpecResolver, bool) {
	if r.j.Type != job.BlockhashStore {
		return nil, false
	}

	return &BlockhashStoreSpecResolver{spec: *r.j.BlockhashStoreSpec}, true
}

func (r *SpecResolver) ToGatewaySpec() (*GatewaySpecResolver, bool) {
	if r.j.Type != job.Gateway {
		return nil, false
	}

	return &GatewaySpecResolver{spec: *r.j.GatewaySpec}, true
}

type VRFSpecResolver struct {
	spec job.VRFSpec
}

// MinIncomingConfirmations resolves the spec's min incoming confirmations.
func (r *VRFSpecResolver) MinIncomingConfirmations() int32 {
	return int32(r.spec.MinIncomingConfirmations)
}

// CoordinatorAddress resolves the spec's coordinator address.
func (r *VRFSpecResolver) CoordinatorAddress() string {
	return r.spec.CoordinatorAddress.String()
}

// CreatedAt resolves the spec's created at timestamp.
func (r *VRFSpecResolver) CreatedAt() graphql.Time {
	return graphql.Time{Time: r.spec.CreatedAt}
}

// EVMChainID resolves the spec's evm chain id.
func (r *VRFSpecResolver) EVMChainID() *string {
	if r.spec.EVMChainID == nil {
		return nil
	}

	chainID := r.spec.EVMChainID.String()

	return &chainID
}

// FromAddresses resolves the spec's from addresses.
func (r *VRFSpecResolver) FromAddresses() *[]string {
	if len(r.spec.FromAddresses) == 0 {
		return nil
	}

	var addresses []string
	for _, a := range r.spec.FromAddresses {
		addresses = append(addresses, a.Address().String())
	}
	return &addresses
}

// PollPeriod resolves the spec's poll period.
func (r *VRFSpecResolver) PollPeriod() string {
	return r.spec.PollPeriod.String()
}

// PublicKey resolves the spec's public key.
func (r *VRFSpecResolver) PublicKey() string {
	return r.spec.PublicKey.String()
}

// RequestedConfsDelay resolves the spec's requested conf delay.
func (r *VRFSpecResolver) RequestedConfsDelay() int32 {
	// GraphQL doesn't support 64 bit integers, so we have to cast.
	return int32(r.spec.RequestedConfsDelay)
}

// RequestTimeout resolves the spec's request timeout.
func (r *VRFSpecResolver) RequestTimeout() string {
	return r.spec.RequestTimeout.String()
}

// BatchCoordinatorAddress resolves the spec's batch coordinator address.
func (r *VRFSpecResolver) BatchCoordinatorAddress() *string {
	if r.spec.BatchCoordinatorAddress == nil {
		return nil
	}
	addr := r.spec.BatchCoordinatorAddress.String()
	return &addr
}

// BatchFulfillmentEnabled resolves the spec's batch fulfillment enabled flag.
func (r *VRFSpecResolver) BatchFulfillmentEnabled() bool {
	return r.spec.BatchFulfillmentEnabled
}

// BatchFulfillmentGasMultiplier resolves the spec's batch fulfillment gas multiplier.
func (r *VRFSpecResolver) BatchFulfillmentGasMultiplier() float64 {
	return float64(r.spec.BatchFulfillmentGasMultiplier)
}

// CustomRevertsPipelineEnabled resolves the spec's custom reverts pipeline enabled flag.
func (r *VRFSpecResolver) CustomRevertsPipelineEnabled() *bool {
	return &r.spec.CustomRevertsPipelineEnabled
}

// ChunkSize resolves the spec's chunk size.
func (r *VRFSpecResolver) ChunkSize() int32 {
	return int32(r.spec.ChunkSize)
}

// BackoffInitialDelay resolves the spec's backoff initial delay.
func (r *VRFSpecResolver) BackoffInitialDelay() string {
	return r.spec.BackoffInitialDelay.String()
}

// BackoffMaxDelay resolves the spec's backoff max delay.
func (r *VRFSpecResolver) BackoffMaxDelay() string {
	return r.spec.BackoffMaxDelay.String()
}

// GasLanePrice resolves the spec's gas lane price.
func (r *VRFSpecResolver) GasLanePrice() *string {
	if r.spec.GasLanePrice == nil {
		return nil
	}
	gasLanePriceGWei := r.spec.GasLanePrice.String()
	return &gasLanePriceGWei
}

// VRFOwnerAddress resolves the spec's vrf owner address.
func (r *VRFSpecResolver) VRFOwnerAddress() *string {
	if r.spec.VRFOwnerAddress == nil {
		return nil
	}
	vrfOwnerAddress := r.spec.VRFOwnerAddress.String()
	return &vrfOwnerAddress
}

type WebhookSpecResolver struct {
	spec job.WebhookSpec
}

// CreatedAt resolves the spec's created at timestamp.
func (r *WebhookSpecResolver) CreatedAt() graphql.Time {
	return graphql.Time{Time: r.spec.CreatedAt}
}

// BlockhashStoreSpecResolver exposes the job parameters for a BlockhashStoreSpec.
type BlockhashStoreSpecResolver struct {
	spec job.BlockhashStoreSpec
}

// CoordinatorV1Address returns the address of the V1 Coordinator, if any.
func (b *BlockhashStoreSpecResolver) CoordinatorV1Address() *string {
	if b.spec.CoordinatorV1Address == nil {
		return nil
	}
	addr := b.spec.CoordinatorV1Address.String()
	return &addr
}

// CoordinatorV2Address returns the address of the V2 Coordinator, if any.
func (b *BlockhashStoreSpecResolver) CoordinatorV2Address() *string {
	if b.spec.CoordinatorV2Address == nil {
		return nil
	}
	addr := b.spec.CoordinatorV2Address.String()
	return &addr
}

// CoordinatorV2PlusAddress returns the address of the V2Plus Coordinator, if any.
func (b *BlockhashStoreSpecResolver) CoordinatorV2PlusAddress() *string {
	if b.spec.CoordinatorV2PlusAddress == nil {
		return nil
	}
	addr := b.spec.CoordinatorV2PlusAddress.String()
	return &addr
}

// WaitBlocks returns the job's WaitBlocks param.
func (b *BlockhashStoreSpecResolver) WaitBlocks() int32 {
	return b.spec.WaitBlocks
}

// LookbackBlocks returns the job's LookbackBlocks param.
func (b *BlockhashStoreSpecResolver) LookbackBlocks() int32 {
	return b.spec.LookbackBlocks
}

// HeartbeatPeriod returns the job's HeartbeatPeriod param.
func (b *BlockhashStoreSpecResolver) HeartbeatPeriod() string {
	return b.spec.HeartbeatPeriod.String()
}

// BlockhashStoreAddress returns the job's BlockhashStoreAddress param.
func (b *BlockhashStoreSpecResolver) BlockhashStoreAddress() string {
	return b.spec.BlockhashStoreAddress.String()
}

// TrustedBlockhashStoreAddress returns the address of the job's TrustedBlockhashStoreAddress, if any.
func (b *BlockhashStoreSpecResolver) TrustedBlockhashStoreAddress() *string {
	if b.spec.TrustedBlockhashStoreAddress == nil {
		return nil
	}
	addr := b.spec.TrustedBlockhashStoreAddress.String()
	return &addr
}

// BatchBlockhashStoreAddress returns the job's BatchBlockhashStoreAddress param.
func (b *BlockhashStoreSpecResolver) TrustedBlockhashStoreBatchSize() int32 {
	return b.spec.TrustedBlockhashStoreBatchSize
}

// PollPeriod return's the job's PollPeriod param.
func (b *BlockhashStoreSpecResolver) PollPeriod() string {
	return b.spec.PollPeriod.String()
}

// RunTimeout return's the job's RunTimeout param.
func (b *BlockhashStoreSpecResolver) RunTimeout() string {
	return b.spec.RunTimeout.String()
}

// EVMChainID returns the job's EVMChainID param.
func (b *BlockhashStoreSpecResolver) EVMChainID() *string {
	chainID := b.spec.EVMChainID.String()
	return &chainID
}

// FromAddress returns the job's FromAddress param, if any.
func (b *BlockhashStoreSpecResolver) FromAddresses() *[]string {
	if b.spec.FromAddresses == nil {
		return nil
	}
	var addresses []string
	for _, a := range b.spec.FromAddresses {
		addresses = append(addresses, a.Address().String())
	}
	return &addresses
}

// CreatedAt resolves the spec's created at timestamp.
func (b *BlockhashStoreSpecResolver) CreatedAt() graphql.Time {
	return graphql.Time{Time: b.spec.CreatedAt}
}

type GatewaySpecResolver struct {
	spec job.GatewaySpec
}

func (r *GatewaySpecResolver) ID() graphql.ID {
	return graphql.ID(stringutils.FromInt32(r.spec.ID))
}

func (r *GatewaySpecResolver) GatewayConfig() gqlscalar.Map {
	return gqlscalar.Map(r.spec.GatewayConfig)
}

func (r *GatewaySpecResolver) CreatedAt() graphql.Time {
	return graphql.Time{Time: r.spec.CreatedAt}
}
