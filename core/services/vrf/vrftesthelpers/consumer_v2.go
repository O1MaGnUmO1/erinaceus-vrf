package vrftesthelpers

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/smartcontractkit/chainlink/v2/core/gethwrappers/generated/vrf_consumer_v2"
	"github.com/smartcontractkit/chainlink/v2/core/gethwrappers/generated/vrf_malicious_consumer_v2"
	"github.com/smartcontractkit/chainlink/v2/core/gethwrappers/generated/vrfv2_reverting_example"
)

var (
	_ VRFConsumerContract = (*vrfConsumerContract)(nil)
)

// VRFConsumerContract is the common interface implemented by
// the example contracts used for the integration tests.
type VRFConsumerContract interface {
	CreateSubscriptionAndFund(opts *bind.TransactOpts, fundingJuels *big.Int) (*gethtypes.Transaction, error)
	CreateSubscriptionAndFundNative(opts *bind.TransactOpts, fundingAmount *big.Int) (*gethtypes.Transaction, error)
	SSubId(opts *bind.CallOpts) (*big.Int, error)
	SRequestId(opts *bind.CallOpts) (*big.Int, error)
	RequestRandomness(opts *bind.TransactOpts, keyHash [32]byte, subID *big.Int, minReqConfs uint16, callbackGasLimit uint32, numWords uint32, payInEth bool) (*gethtypes.Transaction, error)
	SRandomWords(opts *bind.CallOpts, randomwordIdx *big.Int) (*big.Int, error)
	TopUpSubscription(opts *bind.TransactOpts, amount *big.Int) (*gethtypes.Transaction, error)
	TopUpSubscriptionNative(opts *bind.TransactOpts, amount *big.Int) (*gethtypes.Transaction, error)
	SGasAvailable(opts *bind.CallOpts) (*big.Int, error)
	UpdateSubscription(opts *bind.TransactOpts, consumers []common.Address) (*gethtypes.Transaction, error)
	SetSubID(opts *bind.TransactOpts, subID *big.Int) (*gethtypes.Transaction, error)
}

type ConsumerType string

const (
	VRFConsumerV2           ConsumerType = "VRFConsumerV2"
	VRFV2PlusConsumer       ConsumerType = "VRFV2PlusConsumer"
	MaliciousConsumer       ConsumerType = "MaliciousConsumer"
	MaliciousConsumerPlus   ConsumerType = "MaliciousConsumerPlus"
	RevertingConsumer       ConsumerType = "RevertingConsumer"
	RevertingConsumerPlus   ConsumerType = "RevertingConsumerPlus"
	UpgradeableConsumer     ConsumerType = "UpgradeableConsumer"
	UpgradeableConsumerPlus ConsumerType = "UpgradeableConsumerPlus"
)

type vrfConsumerContract struct {
	consumerType      ConsumerType
	vrfConsumerV2     *vrf_consumer_v2.VRFConsumerV2
	maliciousConsumer *vrf_malicious_consumer_v2.VRFMaliciousConsumerV2
	revertingConsumer *vrfv2_reverting_example.VRFV2RevertingExample
}

func NewVRFConsumerV2(consumer *vrf_consumer_v2.VRFConsumerV2) *vrfConsumerContract {
	return &vrfConsumerContract{
		consumerType:  VRFConsumerV2,
		vrfConsumerV2: consumer,
	}
}

func NewMaliciousConsumer(consumer *vrf_malicious_consumer_v2.VRFMaliciousConsumerV2) *vrfConsumerContract {
	return &vrfConsumerContract{
		consumerType:      MaliciousConsumer,
		maliciousConsumer: consumer,
	}
}

func NewRevertingConsumer(consumer *vrfv2_reverting_example.VRFV2RevertingExample) *vrfConsumerContract {
	return &vrfConsumerContract{
		consumerType:      RevertingConsumer,
		revertingConsumer: consumer,
	}
}

func (c *vrfConsumerContract) CreateSubscriptionAndFund(opts *bind.TransactOpts, fundingJuels *big.Int) (*gethtypes.Transaction, error) {
	if c.consumerType == VRFConsumerV2 {
		return c.vrfConsumerV2.CreateSubscriptionAndFund(opts, fundingJuels)
	}
	if c.consumerType == MaliciousConsumer {
		return c.maliciousConsumer.CreateSubscriptionAndFund(opts, fundingJuels)
	}
	if c.consumerType == RevertingConsumer {
		return c.revertingConsumer.CreateSubscriptionAndFund(opts, fundingJuels)
	}
	return nil, errors.New("CreateSubscriptionAndFund is not supported")
}

func (c *vrfConsumerContract) SSubId(opts *bind.CallOpts) (*big.Int, error) {
	if c.consumerType == VRFConsumerV2 {
		subID, err := c.vrfConsumerV2.SSubId(opts)
		if err != nil {
			return nil, err
		}
		return new(big.Int).SetUint64(subID), nil
	}
	if c.consumerType == RevertingConsumer {
		subID, err := c.revertingConsumer.SSubId(opts)
		if err != nil {
			return nil, err
		}
		return new(big.Int).SetUint64(subID), nil
	}
	return nil, errors.New("SSubId is not supported")
}

func (c *vrfConsumerContract) SRequestId(opts *bind.CallOpts) (*big.Int, error) {
	if c.consumerType == VRFConsumerV2 {
		return c.vrfConsumerV2.SRequestId(opts)
	}
	if c.consumerType == MaliciousConsumer {
		return c.maliciousConsumer.SRequestId(opts)
	}
	if c.consumerType == RevertingConsumer {
		return c.revertingConsumer.SRequestId(opts)
	}
	return nil, errors.New("SRequestId is not supported")
}

func (c *vrfConsumerContract) RequestRandomness(opts *bind.TransactOpts, keyHash [32]byte, subID *big.Int, minReqConfs uint16, callbackGasLimit uint32, numWords uint32, payInEth bool) (*gethtypes.Transaction, error) {
	if payInEth {
		return nil, errors.New("eth payment not supported")
	}
	if c.consumerType == VRFConsumerV2 {
		return c.vrfConsumerV2.RequestRandomness(opts, keyHash, subID.Uint64(), minReqConfs, callbackGasLimit, numWords)
	}
	if c.consumerType == MaliciousConsumer {
		return c.maliciousConsumer.RequestRandomness(opts, keyHash)
	}

	if c.consumerType == RevertingConsumer {
		return c.revertingConsumer.RequestRandomness(opts, keyHash, subID.Uint64(), minReqConfs, callbackGasLimit, numWords)
	}
	return nil, errors.New("RequestRandomness is not supported")
}

func (c *vrfConsumerContract) SRandomWords(opts *bind.CallOpts, randomwordIdx *big.Int) (*big.Int, error) {
	if c.consumerType == VRFConsumerV2 {
		return c.vrfConsumerV2.SRandomWords(opts, randomwordIdx)
	}

	return nil, errors.New("SRandomWords is not supported")
}

func (c *vrfConsumerContract) TopUpSubscription(opts *bind.TransactOpts, fundingJuels *big.Int) (*gethtypes.Transaction, error) {
	if c.consumerType == VRFConsumerV2 {
		return c.vrfConsumerV2.TopUpSubscription(opts, fundingJuels)
	}
	if c.consumerType == RevertingConsumer {
		return c.revertingConsumer.TopUpSubscription(opts, fundingJuels)
	}
	return nil, errors.New("TopUpSubscription is not supported")
}

func (c *vrfConsumerContract) SGasAvailable(opts *bind.CallOpts) (*big.Int, error) {
	if c.consumerType == VRFConsumerV2 {
		return c.vrfConsumerV2.SGasAvailable(opts)
	}
	return nil, errors.New("SGasAvailable is not supported")
}

func (c *vrfConsumerContract) UpdateSubscription(opts *bind.TransactOpts, consumers []common.Address) (*gethtypes.Transaction, error) {
	if c.consumerType == VRFConsumerV2 {
		return c.vrfConsumerV2.UpdateSubscription(opts, consumers)
	}
	return nil, errors.New("UpdateSubscription is not supported")
}

func (c *vrfConsumerContract) SetSubID(opts *bind.TransactOpts, subID *big.Int) (*gethtypes.Transaction, error) {
	return nil, errors.New("SetSubID is not supported")
}

func (c *vrfConsumerContract) CreateSubscriptionAndFundNative(opts *bind.TransactOpts, fundingAmount *big.Int) (*gethtypes.Transaction, error) {
	if c.consumerType == VRFV2PlusConsumer {
		// copy object to not mutate original opts
		o := *opts
		o.Value = fundingAmount
		return c.vrfConsumerV2.CreateSubscriptionAndFund(opts, o.Value)
	}
	return nil, errors.New("CreateSubscriptionAndFundNative is not supported")
}

func (c *vrfConsumerContract) TopUpSubscriptionNative(opts *bind.TransactOpts, amount *big.Int) (*gethtypes.Transaction, error) {
	if c.consumerType == VRFV2PlusConsumer {
		// copy object to not mutate original opts
		o := *opts
		o.Value = amount
		return c.vrfConsumerV2.TopUpSubscription(opts, o.Value)
	}
	return nil, errors.New("TopUpSubscriptionNative is not supported")
}
