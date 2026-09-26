package unit_test

import (
	"math"
	"testing"

	"github.com/tayi-ai/arandu-fleet/host"
)

func TestRAMEstimatePreservesHistoricalSixAndFortyTwoRankResults(t *testing.T) {
	// Historical numeric fixture only. The estimator has no model constants.
	budget := host.RAMBudget{
		ShardedStateBytes: 22673533888 * 2, LocalWorld: 2,
		LoaderReserveBytes: 6 << 30, NonShardedMarginBytes: 5 << 30,
	}
	for world, want := range map[int]int64{6: 26926849323, 42: 13970544244} {
		budget.World = world
		got, err := host.EstimateRAM(budget)
		if err != nil || got.RequiredBytes != want {
			t.Fatalf("world %d: required=%d, err=%v; want %d", world, got.RequiredBytes, err, want)
		}
		if got.LoaderReserveBytes != budget.LoaderReserveBytes || got.NonShardedMarginBytes != budget.NonShardedMarginBytes {
			t.Fatal("fixed host reserves were scaled with the process world")
		}
		if declined, err := host.AdmitRAM(budget, want-1); err == nil || declined.RequiredBytes != want || declined.AvailableBytes != want-1 {
			t.Fatalf("insufficient capacity was not reported with its estimate: %+v, %v", declined, err)
		}
		if admitted, err := host.AdmitRAM(budget, want); err != nil || admitted.AvailableBytes != want {
			t.Fatalf("exact capacity was refused: %+v, %v", admitted, err)
		}
	}
}

func TestRAMEstimateRoundsUpWithoutIntermediateOverflow(t *testing.T) {
	for _, test := range []struct {
		state        int64
		world, local int
		want         int64
	}{
		{42, 6, 2, 14},
		{43, 6, 2, 15},
		{1, 42, 1, 1},
		{math.MaxInt64, 1, 1, math.MaxInt64},
		{math.MaxInt64, 4, 3, 6917529027641081856},
		{math.MaxInt64, int(^uint(0) >> 1), int(^uint(0) >> 1), math.MaxInt64},
	} {
		budget := host.RAMBudget{ShardedStateBytes: test.state, World: test.world, LocalWorld: test.local}
		got, err := host.EstimateRAM(budget)
		if err != nil || got.ShardedHostBytes != test.want || got.RequiredBytes != test.want {
			t.Fatalf("budget %+v: estimate=%+v, err=%v; want %d", budget, got, err, test.want)
		}
	}
}

func TestRAMEstimateRejectsInvalidBudgetsAndFinalOverflow(t *testing.T) {
	valid := host.RAMBudget{ShardedStateBytes: 1, World: 1, LocalWorld: 1}
	for name, modify := range map[string]func(*host.RAMBudget){
		"zero state":       func(b *host.RAMBudget) { b.ShardedStateBytes = 0 },
		"negative state":   func(b *host.RAMBudget) { b.ShardedStateBytes = -1 },
		"zero world":       func(b *host.RAMBudget) { b.World = 0 },
		"negative world":   func(b *host.RAMBudget) { b.World = -1 },
		"zero local":       func(b *host.RAMBudget) { b.LocalWorld = 0 },
		"negative local":   func(b *host.RAMBudget) { b.LocalWorld = -1 },
		"excessive local":  func(b *host.RAMBudget) { b.LocalWorld = 2 },
		"negative reserve": func(b *host.RAMBudget) { b.LoaderReserveBytes = -1 },
		"negative margin":  func(b *host.RAMBudget) { b.NonShardedMarginBytes = -1 },
		"reserve overflow": func(b *host.RAMBudget) { b.LoaderReserveBytes = math.MaxInt64 },
		"margin overflow":  func(b *host.RAMBudget) { b.NonShardedMarginBytes = math.MaxInt64 },
		"combined overflow": func(b *host.RAMBudget) {
			b.LoaderReserveBytes, b.NonShardedMarginBytes = math.MaxInt64-1, 1
		},
	} {
		t.Run(name, func(t *testing.T) {
			budget := valid
			modify(&budget)
			if _, err := host.EstimateRAM(budget); err == nil {
				t.Fatalf("invalid budget admitted: %+v", budget)
			}
			if _, err := host.AdmitRAM(budget, math.MaxInt64); err == nil {
				t.Fatalf("capacity admission ignored an invalid budget: %+v", budget)
			}
		})
	}
	if _, err := host.AdmitRAM(valid, -1); err == nil {
		t.Fatal("negative measured capacity admitted")
	}
	if _, err := host.AdmitRAM(valid, 0); err == nil {
		t.Fatal("zero capacity admitted for a positive requirement")
	}
}
