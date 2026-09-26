package host

import (
	"errors"
	"fmt"
	"math"
	"math/bits"
)

// RAMBudget declares state shared evenly across a process world and the fixed
// per-host reserves. The caller supplies all byte counts, including any state
// copies; this type neither inspects nor interprets a model.
type RAMBudget struct {
	ShardedStateBytes     int64
	World                 int
	LocalWorld            int
	LoaderReserveBytes    int64
	NonShardedMarginBytes int64
}

// RAMAdmission records a host's rounded-up share plus its fixed reserves.
// AvailableBytes is populated only by AdmitRAM, not by EstimateRAM.
type RAMAdmission struct {
	AvailableBytes        int64
	RequiredBytes         int64
	ShardedHostBytes      int64
	LoaderReserveBytes    int64
	NonShardedMarginBytes int64
}

// EstimateRAM computes ceil(ShardedStateBytes * LocalWorld / World), then adds
// the two per-host reserves without scaling them. Invalid budgets and totals
// exceeding int64 bytes are refused.
func EstimateRAM(budget RAMBudget) (RAMAdmission, error) {
	if budget.ShardedStateBytes <= 0 || budget.World < 1 || budget.LocalWorld < 1 ||
		budget.LocalWorld > budget.World || budget.LoaderReserveBytes < 0 || budget.NonShardedMarginBytes < 0 {
		return RAMAdmission{}, errors.New("host: invalid RAM budget")
	}
	// A 128-bit intermediate preserves valid estimates when the product exceeds
	// int64. LocalWorld <= World guarantees a quotient no larger than the state
	// and a high word below the divisor, so Div64 cannot overflow.
	high, low := bits.Mul64(uint64(budget.ShardedStateBytes), uint64(budget.LocalWorld))
	share, remainder := bits.Div64(high, low, uint64(budget.World))
	if remainder != 0 {
		share++
	}
	sharded := int64(share)
	if budget.LoaderReserveBytes > math.MaxInt64-sharded {
		return RAMAdmission{}, errors.New("host: RAM requirement overflows int64")
	}
	required := sharded + budget.LoaderReserveBytes
	if budget.NonShardedMarginBytes > math.MaxInt64-required {
		return RAMAdmission{}, errors.New("host: RAM requirement overflows int64")
	}
	return RAMAdmission{
		RequiredBytes:         required + budget.NonShardedMarginBytes,
		ShardedHostBytes:      sharded,
		LoaderReserveBytes:    budget.LoaderReserveBytes,
		NonShardedMarginBytes: budget.NonShardedMarginBytes,
	}, nil
}

// AdmitRAM compares an explicit measured capacity with the estimated budget.
// It returns the estimate alongside insufficient-capacity errors for reporting.
func AdmitRAM(budget RAMBudget, availableBytes int64) (RAMAdmission, error) {
	admission, err := EstimateRAM(budget)
	if err != nil {
		return RAMAdmission{}, err
	}
	admission.AvailableBytes = availableBytes
	if availableBytes < 0 {
		return admission, errors.New("host: available RAM cannot be negative")
	}
	if availableBytes < admission.RequiredBytes {
		return admission, fmt.Errorf("host: RAM admission failed: available=%d bytes required=%d bytes", availableBytes, admission.RequiredBytes)
	}
	return admission, nil
}
