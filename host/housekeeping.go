package host

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Housekeeping is what a node is allowed to clean.
type Housekeeping struct {
	// Programs are the executables this project starts. A process is a candidate
	// only when its name is on this list.
	//
	// Matching by name and not by GPU occupancy is the whole safety argument.
	// `nvidia-smi` will happily name the neighbour's pid, and a cleanup that
	// killed whatever was holding memory would take the client's service down and
	// report success.
	Programs []string
	// ScriptInterpreters optionally allow Programs to match argv[1] of these
	// exact interpreter names. Later arguments never identify a program. No
	// interpreter is assumed by the module; an empty list matches executables only.
	ScriptInterpreters []string
	// Roots are the directories whose contents may be removed. A path that does
	// not resolve under one of these is refused rather than skipped, because a
	// silent skip in a cleanup reads as "there was nothing there".
	Roots []string
	// Patterns name what may go inside those roots. They are what regenerates
	// cheaply or has already served its purpose, and removing them costs a
	// caller nothing it will notice.
	Patterns []string
	// DeepPatterns are what regenerates expensively. They are removed only when
	// a caller asks, because a node cleaned of them takes half an hour to build
	// again -- and a cleanup that quietly cost half an hour is a cleanup nobody
	// runs after the second time.
	DeepPatterns []string
	// Grace is how long a process is given to exit after SIGTERM before the
	// group is killed. A trainer interrupted mid-write leaves a half-written
	// adapter, and the file is renamed into place at the end precisely so that
	// window is short -- but it is not zero.
	Grace time.Duration
	// VRAMFraction is the allocation fraction for the optional post-cleanup
	// measurement. Zero disables measurement; positive values must be in (0, 1].
	VRAMFraction float64
}

// Cleanup is what one pass found and did.
type Cleanup struct {
	Signalled []Process `json:"signalled"`
	Killed    []Process `json:"killed"`
	Removed   []string  `json:"removed"`
	// FreedBytes is what the removals gave back.
	FreedBytes int64 `json:"freed_bytes"`
	// Refused records paths and processes the guards would not touch, and why.
	// A cleanup that silently declined to do something reads afterwards as a
	// cleanup that had nothing to do.
	Refused []string `json:"refused,omitempty"`
	// Cards is what the driver reported once the processes were gone.
	Cards []GPU `json:"cards,omitempty"`
}

// Process is one of ours, as the system reported it.
type Process struct {
	PID     int    `json:"pid"`
	Program string `json:"program"`
	Command string `json:"command"`
}

// FindOurProcesses lists the running programs this installation started.
//
// It reads /proc and matches argv[0], or argv[1] for an explicitly configured
// script interpreter. Arbitrary arguments never identify an owned process.
func (h Housekeeping) FindOurProcesses() ([]Process, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	self := os.Getpid()
	// The processes this one is running inside. Signalling an ancestor kills the
	// shell, the service, or the container's init that is hosting this very
	// command, and the cleanup dies half done -- or takes the host's service with
	// it. Two guards rather than one, because this one failure mode restarts a
	// container.
	ancestors := ancestorsOf(self)
	// The group this process belongs to. Signals go to the group and not the pid,
	// because a launcher exits before the workers it spawned -- but a process we
	// started from the same shell shares our group, and signalling it kills this
	// command too.
	ourGroup, groupErr := syscall.Getpgid(self)
	wanted := map[string]bool{}
	for _, program := range h.Programs {
		wanted[program] = true
	}
	interpreters := map[string]bool{}
	for _, program := range h.ScriptInterpreters {
		interpreters[program] = true
	}
	var found []Process
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid == self {
			continue
		}
		// Never the container's init. Whatever it is running, stopping it stops
		// the machine this command is trying to leave clean.
		if pid == 1 || ancestors[pid] {
			continue
		}
		// And never anything sharing this process's group, for the same reason at
		// a smaller scale.
		if groupErr == nil {
			if group, err := syscall.Getpgid(pid); err == nil && group == ourGroup {
				continue
			}
		}
		command, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "cmdline"))
		if err != nil || len(command) == 0 {
			continue
		}
		parts := strings.Split(strings.TrimRight(string(command), "\x00"), "\x00")
		program := matchedProgram(parts, wanted, interpreters)
		if program == "" {
			continue
		}
		found = append(found, Process{PID: pid, Program: program, Command: strings.Join(parts, " ")})
	}
	sort.Slice(found, func(i, j int) bool { return found[i].PID < found[j].PID })
	return found, nil
}

func matchedProgram(parts []string, programs, interpreters map[string]bool) string {
	if len(parts) == 0 || parts[0] == "" {
		return ""
	}
	program := filepath.Base(parts[0])
	if programs[program] {
		return program
	}
	if interpreters[program] && len(parts) > 1 && parts[1] != "" && !strings.HasPrefix(parts[1], "-") {
		script := filepath.Base(parts[1])
		if programs[script] {
			return script
		}
	}
	return ""
}

// Stop asks this installation's processes to exit, then insists.
//
// The signal goes to the process group and not to the pid. A launcher spawns one
// worker per card and exits before them, so signalling the launcher alone leaves
// the workers holding the memory -- which is exactly the failure this exists to
// prevent, reproduced by the cleanup meant to prevent it.
func (h Housekeeping) Stop(ctx context.Context) (Cleanup, error) {
	result := Cleanup{}
	running, err := h.FindOurProcesses()
	if err != nil {
		return result, err
	}
	if len(running) == 0 {
		return result, nil
	}
	for _, process := range running {
		if err := signalGroup(process.PID, syscall.SIGTERM); err != nil {
			result.Refused = append(result.Refused, fmt.Sprintf("pid %d (%s): %v", process.PID, process.Program, err))
			continue
		}
		result.Signalled = append(result.Signalled, process)
	}

	// The grace is for the writers. A trainer between the temporary file and the
	// rename leaves nothing behind if it is allowed to finish the rename, and a
	// half-written adapter if it is not.
	deadline := time.Now().Add(h.Grace)
	for time.Now().Before(deadline) {
		still, err := h.FindOurProcesses()
		if err != nil || len(still) == 0 {
			break
		}
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}

	remaining, err := h.FindOurProcesses()
	if err != nil {
		return result, err
	}
	for _, process := range remaining {
		if err := signalGroup(process.PID, syscall.SIGKILL); err != nil {
			result.Refused = append(result.Refused, fmt.Sprintf("pid %d (%s): %v", process.PID, process.Program, err))
			continue
		}
		result.Killed = append(result.Killed, process)
	}
	return result, nil
}

// ancestorsOf walks the parent chain from a pid to the top.
//
// It reads PPid out of /proc rather than asking the runtime, because the chain
// that matters crosses processes this program did not start: the shell, the
// service, the init.
func ancestorsOf(pid int) map[int]bool {
	seen := map[int]bool{}
	for depth := 0; pid > 1 && depth < 64; depth++ {
		status, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "status"))
		if err != nil {
			return seen
		}
		parent := 0
		for _, line := range strings.Split(string(status), "\n") {
			if rest, found := strings.CutPrefix(line, "PPid:"); found {
				parent, _ = strconv.Atoi(strings.TrimSpace(rest))
				break
			}
		}
		if parent <= 0 || seen[parent] {
			return seen
		}
		seen[parent] = true
		pid = parent
	}
	return seen
}

func signalGroup(pid int, signal syscall.Signal) error {
	// The third guard, at the last possible moment. The other two live in the
	// listing, and a caller reaching this directly would bypass both.
	if pid <= 1 {
		return errors.New("host: refusing to signal pid 1; it is the container's init and stopping it stops the node")
	}
	if group, err := syscall.Getpgid(pid); err == nil {
		if ours, err := syscall.Getpgid(os.Getpid()); err == nil && group == ours {
			return errors.New("host: refusing to signal this process's own group; the cleanup would stop itself part way through")
		}
	}
	group, err := syscall.Getpgid(pid)
	if err != nil {
		// A process that ended between the listing and the signal is not an
		// error; it is the outcome this wanted.
		if errors.Is(err, syscall.ESRCH) {
			return nil
		}
		return err
	}
	if err := syscall.Kill(-group, signal); err != nil && !errors.Is(err, syscall.ESRCH) {
		return err
	}
	return nil
}

// Sweep removes what the patterns name, and refuses anything that is not under a
// declared root.
//
// The check is on the resolved path, after symlinks. A pattern that matched a
// link pointing outside the roots would otherwise delete whatever it pointed at,
// and on these hosts that is somebody else's data.
func (h Housekeeping) Sweep(deep bool) (Cleanup, error) {
	result := Cleanup{}
	patterns := h.Patterns
	if deep {
		patterns = append(append([]string{}, h.Patterns...), h.DeepPatterns...)
	}
	// The roots are resolved through symlinks too, and not only the matches.
	// Resolving one side and not the other is how a containment check refuses
	// everything: on a host where /workspace is a link, every match resolves to
	// the target and no target is ever under the unresolved root.
	roots := make([]string, 0, len(h.Roots))
	for _, root := range h.Roots {
		resolved, err := filepath.Abs(root)
		if err != nil {
			return result, err
		}
		if linked, err := filepath.EvalSymlinks(resolved); err == nil {
			resolved = linked
		}
		roots = append(roots, resolved)
	}
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return result, fmt.Errorf("host: the pattern %s is malformed: %w", pattern, err)
		}
		for _, match := range matches {
			resolved, err := filepath.EvalSymlinks(match)
			if err != nil {
				resolved = match
			}
			resolved, err = filepath.Abs(resolved)
			if err != nil {
				result.Refused = append(result.Refused, fmt.Sprintf("%s: %v", match, err))
				continue
			}
			if !underAny(resolved, roots) {
				result.Refused = append(result.Refused, fmt.Sprintf("%s resolves to %s, which is outside every declared root", match, resolved))
				continue
			}
			size, err := treeSize(resolved)
			if err != nil {
				result.Refused = append(result.Refused, fmt.Sprintf("%s: %v", match, err))
				continue
			}
			if err := os.RemoveAll(resolved); err != nil {
				result.Refused = append(result.Refused, fmt.Sprintf("%s: %v", match, err))
				continue
			}
			result.Removed = append(result.Removed, resolved)
			result.FreedBytes += size
		}
	}
	return result, nil
}

// underAny reports whether a path sits inside one of the roots.
//
// The separator matters: without it "/workspace-other" passes a prefix test
// against "/workspace".
func underAny(path string, roots []string) bool {
	for _, root := range roots {
		if path == root || strings.HasPrefix(path, root+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func treeSize(path string) (int64, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return 0, err
	}
	if !info.IsDir() {
		return info.Size(), nil
	}
	total := int64(0)
	err = filepath.Walk(path, func(_ string, entry os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !entry.IsDir() {
			total += entry.Size()
		}
		return nil
	})
	return total, err
}

// Clean stops what is running and sweeps what is left.
//
// The order is not interchangeable. Sweeping first would delete files a running
// trainer still has open, which on Linux frees no space until the process exits
// and in the meantime leaves it writing into a file nothing can find.
func (h Housekeeping) Clean(ctx context.Context, sweep, deep bool) (Cleanup, error) {
	if h.VRAMFraction != 0 && !(h.VRAMFraction > 0 && h.VRAMFraction <= 1) {
		return Cleanup{}, errors.New("host: the VRAM fraction must be zero or in (0, 1]")
	}
	result, err := h.Stop(ctx)
	if err != nil {
		return result, err
	}
	if sweep {
		swept, err := h.Sweep(deep)
		result.Removed = append(result.Removed, swept.Removed...)
		result.Refused = append(result.Refused, swept.Refused...)
		result.FreedBytes += swept.FreedBytes
		if err != nil {
			return result, err
		}
	}
	// The cards are read last, and reading them is the only proof this worked.
	// A cleanup that reports what it signalled has reported an intention.
	if h.VRAMFraction > 0 && nvidiaSMIAvailable() {
		if probe, err := ProbeGPUs(ctx, h.VRAMFraction); err == nil {
			result.Cards = probe.GPUs
		}
	}
	return result, nil
}

// Occupied reports the cards that still hold more than a threshold after a
// cleanup, which is how a caller learns that something outside this
// installation's list is holding memory.
//
// It makes no eligibility or ownership decision about the reported cards.
func (c Cleanup) Occupied(threshold int64) []GPU {
	var busy []GPU
	for _, card := range c.Cards {
		if card.TotalBytes-card.FreeBytes > threshold {
			busy = append(busy, card)
		}
	}
	return busy
}

// nvidiaSMIAvailable reports whether the driver can be reached at all, so a
// cleanup on a host with no card says so rather than reporting zero cards.
func nvidiaSMIAvailable() bool {
	_, err := exec.LookPath("nvidia-smi")
	return err == nil
}

// SignalGroupForTest exposes the guard so a test can prove it refuses.
//
// It sends nothing: signal zero asks the kernel whether the signal could be
// delivered, which exercises the refusal without any chance of a test stopping
// the machine it runs on.
func SignalGroupForTest(pid int) error { return signalGroup(pid, 0) }
