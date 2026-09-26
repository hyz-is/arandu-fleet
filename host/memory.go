package host

import (
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
)

// ReadAvailableRAM reads MemAvailable from an explicitly named Linux meminfo
// file. The kernel labels this field kB but defines it in binary KiB.
func ReadAvailableRAM(path string) (int64, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("reading host RAM metric: %w", err)
	}
	for _, line := range strings.Split(string(body), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 3 || fields[0] != "MemAvailable:" {
			continue
		}
		if fields[2] != "kB" {
			return 0, errors.New("MemAvailable has an unexpected unit")
		}
		kib, parseErr := strconv.ParseInt(fields[1], 10, 64)
		if parseErr != nil || kib < 1 || kib > math.MaxInt64/1024 {
			return 0, errors.New("MemAvailable has an invalid value")
		}
		return kib * 1024, nil
	}
	return 0, errors.New("MemAvailable is absent from host metrics")
}

// CgroupAvailable reads the limit and usage of an explicitly named cgroup v2
// memory controller. An unlimited max returns math.MaxInt64 without a usage read.
func CgroupAvailable(maxPath, currentPath string) (int64, error) {
	maximum, err := os.ReadFile(maxPath)
	if err != nil {
		return 0, errors.New("host: cgroup v2 memory limit unavailable")
	}
	if strings.TrimSpace(string(maximum)) == "max" {
		return math.MaxInt64, nil
	}
	limit, err := strconv.ParseInt(strings.TrimSpace(string(maximum)), 10, 64)
	if err != nil || limit <= 0 {
		return 0, errors.New("host: invalid cgroup memory limit")
	}
	current, err := os.ReadFile(currentPath)
	if err != nil {
		return 0, errors.New("host: cgroup memory usage unavailable")
	}
	used, err := strconv.ParseInt(strings.TrimSpace(string(current)), 10, 64)
	if err != nil || used < 0 || used > limit {
		return 0, errors.New("host: invalid cgroup memory usage")
	}
	return limit - used, nil
}
