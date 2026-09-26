package unit_test

import (
	"encoding/json"
	"math"
	"testing"

	host "github.com/tayi-ai/arandu-fleet/host"
)

const twoCards = `0, Example GPU, 7.5, 15360, 3130
1, Example GPU, 7.5, 15360, 3232
`

func TestTheProbeReadsWhatTheDriverPrinted(t *testing.T) {
	probe := host.Probe{Driver: "550.144.03", VRAMFraction: 0.80}
	gpus, err := host.ParseGPUs(twoCards, probe.VRAMFraction)
	if err != nil {
		t.Fatal(err)
	}
	if len(gpus) != 2 {
		t.Fatalf("the probe read %d cards, want 2", len(gpus))
	}
	first := gpus[0]
	if first.Index != 0 || first.Name != "Example GPU" || first.Capability != "7.5" {
		t.Fatalf("the first card came out as %+v", first)
	}
	// The driver reports MiB and the probe reports bytes, because every other
	// number in this project is in bytes and a unit that changes between files is
	// a unit somebody eventually compares wrongly.
	if first.TotalBytes != 15360<<20 {
		t.Fatalf("total is %d bytes, want %d", first.TotalBytes, 15360<<20)
	}
	// Free, not total, is what decides eligibility: the card is shared, and the
	// neighbour holding memory does not itself decide this node's eligibility.
	if first.FreeBytes != 3130<<20 {
		t.Fatalf("free is %d bytes, want %d", first.FreeBytes, 3130<<20)
	}
	// The ceiling is a fraction of the card, not of what is free: it is what this
	// process may allocate, and the neighbour's usage is not this process's to
	// spend.
	if want := int64(float64(15360<<20) * 0.80); first.CapBytes != want {
		t.Fatalf("cap is %d bytes, want %d", first.CapBytes, want)
	}
}

func TestTheProbeRefusesOutputItCannotAccountFor(t *testing.T) {
	for name, out := range map[string]string{
		"missing a column":   "0, Tesla T10, 7.5, 15360\n",
		"extra column":       "0, Tesla T10, 7.5, 15360, 3130, 42\n",
		"index not a number": "x, Tesla T10, 7.5, 15360, 3130\n",
		"size not a number":  "0, Tesla T10, 7.5, plenty, 3130\n",
	} {
		if _, err := host.ParseGPUsForTest(out, 0.80); err == nil {
			t.Errorf("%s: admitted", name)
		}
	}
}

func TestAFractionOutsideTheUnitIntervalIsRefused(t *testing.T) {
	for _, fraction := range []float64{0, -0.1, 1.5, math.NaN(), math.Inf(1), math.Inf(-1)} {
		if _, err := host.ProbeGPUs(t.Context(), fraction); err == nil {
			t.Errorf("fraction %v was admitted", fraction)
		}
	}
}

func TestTheProbePrintsOneJSONLine(t *testing.T) {
	gpus, err := host.ParseGPUs(twoCards, 0.80)
	if err != nil {
		t.Fatal(err)
	}
	line, err := host.Probe{Driver: "550.144.03", VRAMFraction: 0.80, GPUs: gpus}.JSON()
	if err != nil {
		t.Fatal(err)
	}
	var read host.Probe
	if err := json.Unmarshal([]byte(line), &read); err != nil {
		t.Fatalf("the probe printed something that is not JSON: %v", err)
	}
	if len(read.GPUs) != 2 || read.VRAMFraction != 0.80 {
		t.Fatalf("the line did not survive the round trip: %+v", read)
	}
}
