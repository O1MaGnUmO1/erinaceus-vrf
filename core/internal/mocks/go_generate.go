package mocks

//go:generate mockery --quiet --srcpkg github.com/smartcontractkit/chainlink/v2/core/gethwrappers/generated/flux_aggregator_wrapper --name FluxAggregatorInterface --output . --case=underscore --structname FluxAggregator --filename flux_aggregator.go
//go:generate mockery --quiet --srcpkg github.com/smartcontractkit/chainlink/v2/core/gethwrappers/generated/vrf_coordinator_v2 --name VRFCoordinatorV2Interface --output ../../services/vrf/mocks/ --case=underscore --structname VRFCoordinatorV2Interface --filename vrf_coordinator_v2.go
//go:generate mockery --quiet --srcpkg github.com/smartcontractkit/chainlink/v2/core/gethwrappers/ocr2vrf/generated/vrf_beacon --name VRFBeaconInterface --output ../../services/ocr2/plugins/ocr2vrf/coordinator/mocks --case=underscore --structname VRFBeaconInterface --filename vrf_beacon.go
//go:generate mockery --quiet --srcpkg github.com/smartcontractkit/chainlink/v2/core/gethwrappers/ocr2vrf/generated/vrf_coordinator --name VRFCoordinatorInterface --output ../../services/ocr2/plugins/ocr2vrf/coordinator/mocks --case=underscore --structname VRFCoordinatorInterface --filename vrf_coordinator.go