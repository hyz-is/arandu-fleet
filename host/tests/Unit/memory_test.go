package unit_test

import (
	"math"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/tayi-ai/arandu-fleet/host"
)

func TestReadAvailableRAMUsesKernelUnitsAndRejectsInvalidMeasurements(t *testing.T) {
	for _, test := range []struct {
		name string
		body string
		want int64
	}{
		{"available not total", "MemTotal: 9999999 kB\nMemAvailable: 4194304 kB\n", 4 << 30},
		{"largest convertible", "MemAvailable: " + strconv.FormatInt(math.MaxInt64/1024, 10) + " kB\n", (math.MaxInt64 / 1024) * 1024},
		{"absent", "MemFree: 100 kB\n", 0},
		{"wrong unit", "MemAvailable: 10 MB\n", 0},
		{"negative", "MemAvailable: -1 kB\n", 0},
		{"zero", "MemAvailable: 0 kB\n", 0},
		{"not integer", "MemAvailable: 1.5 kB\n", 0},
		{"conversion overflow", "MemAvailable: " + strconv.FormatInt(math.MaxInt64/1024+1, 10) + " kB\n", 0},
		{"integer overflow", "MemAvailable: 9223372036854775808 kB\n", 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "meminfo")
			if err := os.WriteFile(path, []byte(test.body), 0o600); err != nil {
				t.Fatal(err)
			}
			got, err := host.ReadAvailableRAM(path)
			if test.want == 0 {
				if err == nil {
					t.Fatalf("invalid measurement accepted: %d", got)
				}
			} else if err != nil || got != test.want {
				t.Fatalf("available = %d, %v; want %d", got, err, test.want)
			}
		})
	}
	if _, err := host.ReadAvailableRAM(filepath.Join(t.TempDir(), "absent")); err == nil {
		t.Fatal("missing meminfo accepted")
	}
}

func TestCgroupAvailablePreservesLimitsAndRejectsInvalidUsage(t *testing.T) {
	for _, test := range []struct {
		name, maximum, current string
		want                   int64
		invalid                bool
	}{
		{name: "remaining bytes", maximum: "4096\n", current: "1024\n", want: 3072},
		{name: "empty", maximum: "4096", current: "0", want: 4096},
		{name: "full", maximum: "4096", current: "4096", want: 0},
		{name: "largest limit", maximum: "9223372036854775807", current: "1", want: math.MaxInt64 - 1},
		{name: "unlimited without usage", maximum: "max\n", want: math.MaxInt64},
		{name: "zero limit", maximum: "0", current: "0", invalid: true},
		{name: "negative limit", maximum: "-1", current: "0", invalid: true},
		{name: "overflow limit", maximum: "9223372036854775808", current: "0", invalid: true},
		{name: "negative usage", maximum: "4096", current: "-1", invalid: true},
		{name: "excessive usage", maximum: "4096", current: "4097", invalid: true},
		{name: "overflow usage", maximum: "4096", current: "9223372036854775808", invalid: true},
		{name: "missing usage", maximum: "4096", invalid: true},
		{name: "missing limit", current: "1", invalid: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			maxPath, currentPath := filepath.Join(dir, "memory.max"), filepath.Join(dir, "memory.current")
			for path, body := range map[string]string{maxPath: test.maximum, currentPath: test.current} {
				if body != "" {
					if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
						t.Fatal(err)
					}
				}
			}
			got, err := host.CgroupAvailable(maxPath, currentPath)
			if test.invalid {
				if err == nil {
					t.Fatalf("invalid cgroup measurement accepted: %d", got)
				}
			} else if err != nil || got != test.want {
				t.Fatalf("available = %d, %v; want %d", got, err, test.want)
			}
		})
	}
}
