package host

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// GPU is one card as the driver reports it.
type GPU struct {
	Index int    `json:"index"`
	Name  string `json:"name"`
	// Capability is the compute capability reported by the driver.
	Capability string `json:"capability"`
	// TotalBytes is the card. FreeBytes is what is left on it right now.
	TotalBytes int64 `json:"total_bytes"`
	// FreeBytes is current free memory; the caller decides eligibility.
	FreeBytes int64 `json:"free_bytes"`
	// CapBytes is the ceiling one process may allocate here, from the declared
	// fraction. Without it the run grows until the card is full and whoever
	// starts last takes the OOM.
	CapBytes int64 `json:"cap_bytes"`
}

// Probe is what this host answers about its cards.
type Probe struct {
	Driver       string  `json:"driver"`
	VRAMFraction float64 `json:"vram_fraction"`
	GPUs         []GPU   `json:"gpus"`
}

// gpuQuery is what nvidia-smi is asked for, in order. Bytes are requested as MiB
// because that is the only unit it offers, and converted once here.
const gpuQuery = "index,name,compute_cap,memory.total,memory.free"

// ProbeGPUs asks the driver what this host has.
//
// It refuses rather than reporting zero cards. A host that answers "no GPU" and
// a host whose driver could not be reached look identical to a caller that gets
// an empty list, and the second one is a broken node reporting itself healthy.
func ProbeGPUs(ctx context.Context, fraction float64) (Probe, error) {
	if !(fraction > 0 && fraction <= 1) {
		return Probe{}, fmt.Errorf("host: the VRAM fraction has to be in (0, 1]; got %v", fraction)
	}
	out, err := exec.CommandContext(ctx, "nvidia-smi",
		"--query-gpu="+gpuQuery, "--format=csv,noheader,nounits").Output()
	if err != nil {
		return Probe{}, fmt.Errorf("host: reading the driver with nvidia-smi: %w", err)
	}
	gpus, err := ParseGPUs(string(out), fraction)
	if err != nil {
		return Probe{}, err
	}
	if len(gpus) == 0 {
		return Probe{}, fmt.Errorf("host: the driver reported no card; check NVIDIA_VISIBLE_DEVICES")
	}
	driver, err := exec.CommandContext(ctx, "nvidia-smi",
		"--query-gpu=driver_version", "--format=csv,noheader").Output()
	if err != nil {
		return Probe{}, fmt.Errorf("host: reading the driver version: %w", err)
	}
	version := strings.TrimSpace(string(driver))
	if i := strings.IndexByte(version, '\n'); i >= 0 {
		version = version[:i]
	}
	return Probe{Driver: strings.TrimSpace(version), VRAMFraction: fraction, GPUs: gpus}, nil
}

// ParseGPUs converts the five-column driver CSV to byte measurements using the
// caller's allocation fraction. It performs no driver or process access.
func ParseGPUs(out string, fraction float64) ([]GPU, error) {
	if !(fraction > 0 && fraction <= 1) {
		return nil, fmt.Errorf("host: the VRAM fraction has to be in (0, 1]; got %v", fraction)
	}
	var gpus []GPU
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Split(line, ",")
		if len(fields) != 5 {
			return nil, fmt.Errorf("host: the driver printed %d columns for %q, want 5", len(fields), strings.TrimSpace(line))
		}
		for i := range fields {
			fields[i] = strings.TrimSpace(fields[i])
		}
		index, err := strconv.Atoi(fields[0])
		if err != nil {
			return nil, fmt.Errorf("host: the driver printed %q as a card index: %w", fields[0], err)
		}
		total, err := mib(fields[3])
		if err != nil {
			return nil, err
		}
		free, err := mib(fields[4])
		if err != nil {
			return nil, err
		}
		gpus = append(gpus, GPU{
			Index: index, Name: fields[1], Capability: fields[2],
			TotalBytes: total, FreeBytes: free,
			CapBytes: int64(float64(total) * fraction),
		})
	}
	return gpus, nil
}

// mib converts the driver's mebibytes to bytes.
func mib(s string) (int64, error) {
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("host: the driver printed %q where a size in MiB was expected: %w", s, err)
	}
	return v << 20, nil
}

// JSON is what the diagnostics command prints: one line, so a control plane
// reading a node's log can parse it without knowing how long it is.
func (p Probe) JSON() (string, error) {
	body, err := json.Marshal(p)
	return string(body), err
}

// ParseGPUsForTest exposes the parser to the test package.
//
// The parser is the half worth testing and the half that needs no card: it turns
// the driver's own output into the numbers eligibility is decided from. Exposing
// it is what lets that be exercised on a laptop, which is where this is read.
func ParseGPUsForTest(out string, fraction float64) ([]GPU, error) { return ParseGPUs(out, fraction) }
