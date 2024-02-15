package log_test

import (
	"context"
	"math/big"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/O1MaGnUmO1/chainlink-common/pkg/logger"

	evmclimocks "github.com/smartcontractkit/chainlink/v2/core/chains/evm/client/mocks"
	"github.com/smartcontractkit/chainlink/v2/core/chains/evm/log"
	logmocks "github.com/smartcontractkit/chainlink/v2/core/chains/evm/log/mocks"
	evmtypes "github.com/smartcontractkit/chainlink/v2/core/chains/evm/types"
	"github.com/smartcontractkit/chainlink/v2/core/internal/cltest"
	"github.com/smartcontractkit/chainlink/v2/core/internal/testutils"
	"github.com/smartcontractkit/chainlink/v2/core/internal/testutils/evmtest"
	"github.com/smartcontractkit/chainlink/v2/core/services/chainlink"
)

func TestBroadcaster_AwaitsInitialSubscribersOnStartup(t *testing.T) {
	g := gomega.NewWithT(t)

	const blockHeight int64 = 123
	helper := newBroadcasterHelper(t, blockHeight, 1, nil, nil)
	helper.lb.AddDependents(2)

	var listener = helper.newLogListenerWithJob("A")
	helper.register(listener, newMockContract(t), 1)

	helper.start()
	defer helper.stop()

	require.Eventually(t, func() bool { return helper.mockEth.SubscribeCallCount() == 0 }, testutils.WaitTimeout(t), 100*time.Millisecond)
	g.Consistently(func() int32 { return helper.mockEth.SubscribeCallCount() }, 1*time.Second, cltest.DBPollingInterval).Should(gomega.Equal(int32(0)))

	helper.lb.DependentReady()

	require.Eventually(t, func() bool { return helper.mockEth.SubscribeCallCount() == 0 }, testutils.WaitTimeout(t), 100*time.Millisecond)
	g.Consistently(func() int32 { return helper.mockEth.SubscribeCallCount() }, 1*time.Second, cltest.DBPollingInterval).Should(gomega.Equal(int32(0)))

	helper.lb.DependentReady()

	require.Eventually(t, func() bool { return helper.mockEth.SubscribeCallCount() == 1 }, testutils.WaitTimeout(t), 100*time.Millisecond)
	g.Consistently(func() int32 { return helper.mockEth.SubscribeCallCount() }, 1*time.Second, cltest.DBPollingInterval).Should(gomega.Equal(int32(1)))

	helper.unsubscribeAll()

	require.Eventually(t, func() bool { return helper.mockEth.UnsubscribeCallCount() == 1 }, testutils.WaitTimeout(t), 100*time.Millisecond)
	g.Consistently(func() int32 { return helper.mockEth.UnsubscribeCallCount() }, 1*time.Second, cltest.DBPollingInterval).Should(gomega.Equal(int32(1)))
}

func TestBroadcaster_ResubscribesOnAddOrRemoveContract(t *testing.T) {
	testutils.SkipShortDB(t)
	const (
		numConfirmations            = 1
		numContracts                = 3
		blockHeight           int64 = 123
		lastStoredBlockHeight       = blockHeight - 25
	)

	backfillTimes := 2
	expectedCalls := mockEthClientExpectedCalls{
		SubscribeFilterLogs: backfillTimes,
		HeaderByNumber:      backfillTimes,
		FilterLogs:          backfillTimes,
	}

	chchRawLogs := make(chan evmtest.RawSub[types.Log], backfillTimes)
	mockEth := newMockEthClient(t, chchRawLogs, blockHeight, expectedCalls)
	helper := newBroadcasterHelperWithEthClient(t, mockEth.EthClient, cltest.Head(lastStoredBlockHeight), nil)
	helper.mockEth = mockEth

	blockBackfillDepth := helper.config.EVM().BlockBackfillDepth()

	var backfillCount atomic.Int64

	// the first backfill should use the height of last head saved to the db,
	// minus maxNumConfirmations of subscribers and minus blockBackfillDepth
	mockEth.CheckFilterLogs = func(fromBlock int64, toBlock int64) {
		backfillCount.Store(1)
		require.Equal(t, lastStoredBlockHeight-numConfirmations-int64(blockBackfillDepth), fromBlock)
	}

	listener := helper.newLogListenerWithJob("initial")

	helper.register(listener, newMockContract(t), numConfirmations)

	for i := 0; i < numContracts; i++ {
		listener := helper.newLogListenerWithJob("")
		helper.register(listener, newMockContract(t), 1)
	}

	helper.start()
	defer helper.stop()

	require.Eventually(t, func() bool { return helper.mockEth.SubscribeCallCount() == 1 }, testutils.WaitTimeout(t), time.Second)
	gomega.NewWithT(t).Consistently(func() int32 { return helper.mockEth.SubscribeCallCount() }, 1*time.Second, cltest.DBPollingInterval).Should(gomega.Equal(int32(1)))
	gomega.NewWithT(t).Consistently(func() int32 { return helper.mockEth.UnsubscribeCallCount() }, 1*time.Second, cltest.DBPollingInterval).Should(gomega.Equal(int32(0)))

	require.Eventually(t, func() bool { return backfillCount.Load() == 1 }, testutils.WaitTimeout(t), 100*time.Millisecond)
	helper.unsubscribeAll()

	// now the backfill must use the blockBackfillDepth
	mockEth.CheckFilterLogs = func(fromBlock int64, toBlock int64) {
		require.Equal(t, blockHeight-int64(blockBackfillDepth), fromBlock)
		backfillCount.Store(2)
	}

	listenerLast := helper.newLogListenerWithJob("last")
	helper.register(listenerLast, newMockContract(t), 1)

	require.Eventually(t, func() bool { return helper.mockEth.UnsubscribeCallCount() >= 1 }, testutils.WaitTimeout(t), time.Second)
	gomega.NewWithT(t).Consistently(func() int32 { return helper.mockEth.SubscribeCallCount() }, 1*time.Second, cltest.DBPollingInterval).Should(gomega.Equal(int32(2)))
	gomega.NewWithT(t).Consistently(func() int32 { return helper.mockEth.UnsubscribeCallCount() }, 1*time.Second, cltest.DBPollingInterval).Should(gomega.Equal(int32(1)))

	require.Eventually(t, func() bool { return backfillCount.Load() == 2 }, testutils.WaitTimeout(t), time.Second)
}

func TestBroadcaster_BackfillOnNodeStartAndOnReplay(t *testing.T) {
	testutils.SkipShortDB(t)
	const (
		lastStoredBlockHeight       = 100
		blockHeight           int64 = 125
		replayFrom            int64 = 40
	)

	backfillTimes := 2
	expectedCalls := mockEthClientExpectedCalls{
		SubscribeFilterLogs: backfillTimes,
		HeaderByNumber:      backfillTimes,
		FilterLogs:          2,
	}

	chchRawLogs := make(chan evmtest.RawSub[types.Log], backfillTimes)
	mockEth := newMockEthClient(t, chchRawLogs, blockHeight, expectedCalls)
	helper := newBroadcasterHelperWithEthClient(t, mockEth.EthClient, cltest.Head(lastStoredBlockHeight), nil)
	helper.mockEth = mockEth

	maxNumConfirmations := int64(10)

	var backfillCount atomic.Int64

	listener := helper.newLogListenerWithJob("one")
	helper.register(listener, newMockContract(t), uint32(maxNumConfirmations))

	listener2 := helper.newLogListenerWithJob("two")
	helper.register(listener2, newMockContract(t), uint32(2))

	blockBackfillDepth := helper.config.EVM().BlockBackfillDepth()

	// the first backfill should use the height of last head saved to the db,
	// minus maxNumConfirmations of subscribers and minus blockBackfillDepth
	mockEth.CheckFilterLogs = func(fromBlock int64, toBlock int64) {
		times := backfillCount.Add(1) - 1
		if times == 0 {
			require.Equal(t, lastStoredBlockHeight-maxNumConfirmations-int64(blockBackfillDepth), fromBlock)
		} else if times == 1 {
			require.Equal(t, replayFrom, fromBlock)
		}
	}

	func() {
		helper.start()
		defer helper.stop()

		require.Eventually(t, func() bool { return helper.mockEth.SubscribeCallCount() == 1 }, testutils.WaitTimeout(t), time.Second)
		require.Eventually(t, func() bool { return backfillCount.Load() == 1 }, testutils.WaitTimeout(t), time.Second)

		helper.lb.ReplayFromBlock(replayFrom, false)

		require.Eventually(t, func() bool { return backfillCount.Load() >= 2 }, testutils.WaitTimeout(t), time.Second)
	}()

	require.Eventually(t, func() bool { return helper.mockEth.UnsubscribeCallCount() >= 1 }, testutils.WaitTimeout(t), time.Second)
}

func (helper *broadcasterHelper) simulateHeads(t *testing.T, listener, listener2 *simpleLogListener,
	contract1, contract2 *logmocks.AbigenContract, confs uint32, heads []*evmtypes.Head, orm log.ORM, assertBlock *int64, do func()) {
	helper.lb.AddDependents(2)
	helper.start()
	defer helper.stop()
	helper.register(listener, contract1, confs)
	helper.register(listener2, contract2, confs)
	helper.lb.DependentReady()
	helper.lb.DependentReady()

	headsDone := cltest.SimulateIncomingHeads(t, heads, helper.lb)

	if do != nil {
		do()
	}

	<-headsDone

	require.Eventually(t, func() bool {
		blockNum, err := orm.GetPendingMinBlock()
		if !assert.NoError(t, err) {
			return false
		}
		if assertBlock == nil {
			return blockNum == nil
		} else if blockNum == nil {
			return false
		}
		return *assertBlock == *blockNum
	}, testutils.WaitTimeout(t), time.Second)
}

func TestBroadcaster_ShallowBackfillOnNodeStart(t *testing.T) {
	testutils.SkipShortDB(t)
	const (
		lastStoredBlockHeight       = 100
		blockHeight           int64 = 125
		backfillDepth               = 15
	)

	backfillTimes := 1
	expectedCalls := mockEthClientExpectedCalls{
		SubscribeFilterLogs: backfillTimes,
		HeaderByNumber:      backfillTimes,
		FilterLogs:          backfillTimes,
	}

	chchRawLogs := make(chan evmtest.RawSub[types.Log], backfillTimes)
	mockEth := newMockEthClient(t, chchRawLogs, blockHeight, expectedCalls)
	helper := newBroadcasterHelperWithEthClient(t, mockEth.EthClient, cltest.Head(lastStoredBlockHeight), func(c *chainlink.Config, s *chainlink.Secrets) {
		c.EVM[0].BlockBackfillSkip = ptr(true)
		c.EVM[0].BlockBackfillDepth = ptr[uint32](15)
	})
	helper.mockEth = mockEth

	var backfillCount atomic.Int64

	listener := helper.newLogListenerWithJob("one")
	helper.register(listener, newMockContract(t), uint32(10))

	listener2 := helper.newLogListenerWithJob("two")
	helper.register(listener2, newMockContract(t), uint32(2))

	// the backfill does not use the height from DB because BlockBackfillSkip is true
	mockEth.CheckFilterLogs = func(fromBlock int64, toBlock int64) {
		backfillCount.Store(1)
		require.Equal(t, blockHeight-int64(backfillDepth), fromBlock)
	}

	func() {
		helper.start()
		defer helper.stop()

		require.Eventually(t, func() bool { return helper.mockEth.SubscribeCallCount() == 1 }, testutils.WaitTimeout(t), time.Second)
		require.Eventually(t, func() bool { return backfillCount.Load() == 1 }, testutils.WaitTimeout(t), time.Second)
	}()

	require.Eventually(t, func() bool { return helper.mockEth.UnsubscribeCallCount() >= 1 }, testutils.WaitTimeout(t), time.Second)
}

func TestBroadcaster_BackfillInBatches(t *testing.T) {
	testutils.SkipShortDB(t)
	const (
		numConfirmations            = 1
		blockHeight           int64 = 120
		lastStoredBlockHeight       = blockHeight - 29
		backfillTimes               = 1
		batchSize             int64 = 5
		expectedBatches             = 9
	)

	expectedCalls := mockEthClientExpectedCalls{
		SubscribeFilterLogs: backfillTimes,
		HeaderByNumber:      backfillTimes,
		FilterLogs:          expectedBatches,
	}

	chchRawLogs := make(chan evmtest.RawSub[types.Log], backfillTimes)
	mockEth := newMockEthClient(t, chchRawLogs, blockHeight, expectedCalls)
	helper := newBroadcasterHelperWithEthClient(t, mockEth.EthClient, cltest.Head(lastStoredBlockHeight), func(c *chainlink.Config, s *chainlink.Secrets) {
		c.EVM[0].LogBackfillBatchSize = ptr(uint32(batchSize))
	})
	helper.mockEth = mockEth

	blockBackfillDepth := helper.config.EVM().BlockBackfillDepth()

	var backfillCount atomic.Int64

	lggr := logger.Test(t)
	backfillStart := lastStoredBlockHeight - numConfirmations - int64(blockBackfillDepth)
	// the first backfill should start from before the last stored head
	mockEth.CheckFilterLogs = func(fromBlock int64, toBlock int64) {
		times := backfillCount.Add(1) - 1
		lggr.Infof("Log Batch: --------- times %v - %v, %v", times, fromBlock, toBlock)

		if times <= 7 {
			require.Equal(t, backfillStart+batchSize*times, fromBlock)
			require.Equal(t, backfillStart+batchSize*(times+1)-1, toBlock)
		} else {
			// last batch is for a range of 1
			require.Equal(t, int64(120), fromBlock)
			require.Equal(t, int64(120), toBlock)
		}
	}

	listener := helper.newLogListenerWithJob("initial")
	helper.register(listener, newMockContract(t), numConfirmations)
	helper.start()

	defer helper.stop()

	require.Eventually(t, func() bool { return backfillCount.Load() == expectedBatches }, testutils.WaitTimeout(t), time.Second)

	helper.unsubscribeAll()

	require.Eventually(t, func() bool { return helper.mockEth.UnsubscribeCallCount() >= 1 }, testutils.WaitTimeout(t), time.Second)
}

func TestBroadcaster_Register_ResubscribesToMostRecentlySeenBlock(t *testing.T) {
	testutils.SkipShortDB(t)
	const (
		backfillTimes = 1
		blockHeight   = 15
		expectedBlock = 5
	)
	var (
		ethClient = evmclimocks.NewClient(t)
		contract0 = newMockContract(t)
		contract1 = newMockContract(t)
		contract2 = newMockContract(t)
	)
	mockEth := &evmtest.MockEth{EthClient: ethClient}
	chchRawLogs := make(chan evmtest.RawSub[types.Log], backfillTimes)
	chStarted := make(chan struct{})
	ethClient.On("ConfiguredChainID", mock.Anything).Return(&cltest.FixtureChainID)
	ethClient.On("SubscribeFilterLogs", mock.Anything, mock.Anything, mock.Anything).
		Return(
			func(ctx context.Context, q ethereum.FilterQuery, ch chan<- types.Log) ethereum.Subscription {
				defer close(chStarted)
				sub := mockEth.NewSub(t)
				chchRawLogs <- evmtest.NewRawSub(ch, sub.Err())
				return sub
			},
			func(ctx context.Context, q ethereum.FilterQuery, ch chan<- types.Log) error {
				return nil
			},
		).
		Once()

	ethClient.On("SubscribeFilterLogs", mock.Anything, mock.Anything, mock.Anything).
		Return(
			func(ctx context.Context, q ethereum.FilterQuery, ch chan<- types.Log) ethereum.Subscription {
				sub := mockEth.NewSub(t)
				chchRawLogs <- evmtest.NewRawSub(ch, sub.Err())
				return sub
			},
			func(ctx context.Context, q ethereum.FilterQuery, ch chan<- types.Log) error {
				return nil
			},
		).
		Times(3)

	ethClient.On("HeadByNumber", mock.Anything, (*big.Int)(nil)).
		Return(&evmtypes.Head{Number: blockHeight}, nil)

	ethClient.On("FilterLogs", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			query := args.Get(1).(ethereum.FilterQuery)
			require.Equal(t, big.NewInt(expectedBlock), query.FromBlock)
			require.Contains(t, query.Addresses, contract0.Address())
			require.Len(t, query.Addresses, 1)
		}).
		Return(nil, nil).
		Times(backfillTimes)

	ethClient.On("FilterLogs", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			query := args.Get(1).(ethereum.FilterQuery)
			require.Equal(t, big.NewInt(expectedBlock), query.FromBlock)
			require.Contains(t, query.Addresses, contract0.Address())
			require.Contains(t, query.Addresses, contract1.Address())
			require.Len(t, query.Addresses, 2)
		}).
		Return(nil, nil).
		Once()

	ethClient.On("FilterLogs", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			query := args.Get(1).(ethereum.FilterQuery)
			require.Equal(t, big.NewInt(expectedBlock), query.FromBlock)
			require.Contains(t, query.Addresses, contract0.Address())
			require.Contains(t, query.Addresses, contract1.Address())
			require.Contains(t, query.Addresses, contract2.Address())
			require.Len(t, query.Addresses, 3)
		}).
		Return(nil, nil).
		Once()

	helper := newBroadcasterHelperWithEthClient(t, ethClient, nil, nil)
	helper.lb.AddDependents(1)
	helper.start()
	defer helper.stop()

	listener0 := helper.newLogListenerWithJob("0")
	listener1 := helper.newLogListenerWithJob("1")
	listener2 := helper.newLogListenerWithJob("2")

	// Subscribe #0
	helper.register(listener0, contract0, 1)
	defer helper.unsubscribeAll()
	helper.lb.DependentReady()

	// Await startup
	select {
	case <-chStarted:
	case <-time.After(testutils.WaitTimeout(t)):
		t.Fatal("never started")
	}

	select {
	case <-chchRawLogs:
	case <-time.After(testutils.WaitTimeout(t)):
		t.Fatal("did not subscribe")
	}

	// Subscribe #1
	helper.register(listener1, contract1, 1)

	select {
	case <-chchRawLogs:
	case <-time.After(testutils.WaitTimeout(t)):
		t.Fatal("did not subscribe")
	}

	// Subscribe #2
	helper.register(listener2, contract2, 1)

	select {
	case <-chchRawLogs:
	case <-time.After(testutils.WaitTimeout(t)):
		t.Fatal("did not subscribe")
	}

	// ReplayFrom will not lead to backfill because the number is above current height
	helper.lb.ReplayFromBlock(125, false)

	select {
	case <-chchRawLogs:
	case <-time.After(testutils.WaitTimeout(t)):
		t.Fatal("did not subscribe")
	}

	cltest.EventuallyExpectationsMet(t, ethClient, testutils.WaitTimeout(t), time.Second)
}

func pickLogs(allLogs map[uint]types.Log, indices []uint) []types.Log {
	var picked []types.Log
	for _, idx := range indices {
		picked = append(picked, allLogs[idx])
	}
	return picked
}

func requireEqualLogs(t *testing.T, expectedLogs, actualLogs []types.Log) {
	t.Helper()
	require.Equalf(t, len(expectedLogs), len(actualLogs), "log slices are not equal (len %v vs %v): expected(%v), actual(%v)", len(expectedLogs), len(actualLogs), expectedLogs, actualLogs)
	for i := range expectedLogs {
		require.Equalf(t, expectedLogs[i], actualLogs[i], "log slices are not equal (len %v vs %v): expected(%v), actual(%v)", len(expectedLogs), len(actualLogs), expectedLogs, actualLogs)
	}
}

func ptr[T any](t T) *T { return &t }
