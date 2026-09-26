package unit_test

import (
	"os"
	"path/filepath"
	"testing"

	host "github.com/tayi-ai/arandu-fleet/host"
)

func TestASweepRefusesWhatLeavesItsRoots(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "root")
	outside := filepath.Join(dir, "outside")
	for _, d := range []string{root, outside} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// A file that must survive, reached through a link that sits inside the root.
	// The link is the attack: a pattern matches it, and a sweep that removed what
	// a pattern matched would delete the target.
	target := filepath.Join(outside, "someone-elses-data")
	if err := os.WriteFile(target, []byte("not ours"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, "sweep-me")); err != nil {
		t.Skipf("this filesystem refuses symlinks: %v", err)
	}
	// And a real file inside the root, which must go.
	ours := filepath.Join(root, "sweep-mine")
	if err := os.WriteFile(ours, []byte("regenerable"), 0o644); err != nil {
		t.Fatal(err)
	}

	keeper := host.Housekeeping{
		Roots:    []string{root},
		Patterns: []string{filepath.Join(root, "sweep-*")},
	}
	result, err := keeper.Sweep(false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("the sweep followed a link out of its root and removed %s", target)
	}
	if len(result.Refused) == 0 {
		t.Error("the sweep removed nothing outside its root but did not say it had refused anything")
	}
	if _, err := os.Stat(ours); !os.IsNotExist(err) {
		t.Errorf("the sweep left %s, which is inside its root and matched", ours)
	}
	if result.FreedBytes <= 0 {
		t.Errorf("the sweep reported %d bytes freed after removing a file", result.FreedBytes)
	}
}

func TestASweepDoesNotMistakeANeighbouringDirectoryForItsRoot(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "workspace")
	sibling := filepath.Join(dir, "workspace-other")
	for _, d := range []string{root, sibling} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	victim := filepath.Join(sibling, "sweep-me")
	if err := os.WriteFile(victim, []byte("another tenant"), 0o644); err != nil {
		t.Fatal(err)
	}
	keeper := host.Housekeeping{
		Roots: []string{root},
		// The pattern deliberately reaches the sibling. Without the separator in
		// the containment check, "workspace-other" passes a prefix test against
		// "workspace".
		Patterns: []string{filepath.Join(sibling, "sweep-*")},
	}
	result, err := keeper.Sweep(false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(victim); err != nil {
		t.Fatalf("the sweep treated %s as inside %s and removed %s", sibling, root, victim)
	}
	if len(result.Removed) != 0 {
		t.Errorf("the sweep removed %v from outside its root", result.Removed)
	}
	if len(result.Refused) != 1 {
		t.Errorf("the sweep did not report the neighboring match as refused: %+v", result)
	}
}

func TestTheDeepSweepIsOptIn(t *testing.T) {
	dir := t.TempDir()
	shallow := filepath.Join(dir, "cheap")
	expensive := filepath.Join(dir, "costly")
	for _, p := range []string{shallow, expensive} {
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	keeper := host.Housekeeping{
		Roots:        []string{dir},
		Patterns:     []string{shallow},
		DeepPatterns: []string{expensive},
	}
	if _, err := keeper.Sweep(false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(expensive); err != nil {
		t.Fatal("a shallow sweep removed what costs half an hour to rebuild")
	}
	if _, err := os.Stat(shallow); !os.IsNotExist(err) {
		t.Fatal("a shallow sweep left what regenerates in milliseconds")
	}
	if _, err := keeper.Sweep(true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(expensive); !os.IsNotExist(err) {
		t.Fatal("a deep sweep left the expensive tree")
	}
}

func TestOccupiedNamesTheCardsSomethingElseIsHolding(t *testing.T) {
	// After a cleanup the neighbour is still there. What matters is
	// telling its 1.5 GiB apart from a program this installation does not know
	// about, so the threshold is what the caller decides and this only reports.
	result := host.Cleanup{Cards: []host.GPU{
		{Index: 0, TotalBytes: 16 << 30, FreeBytes: 14 << 30}, // 2 GiB in use
		{Index: 1, TotalBytes: 16 << 30, FreeBytes: 4 << 30},  // 12 GiB in use
	}}
	busy := result.Occupied(4 << 30)
	if len(busy) != 1 || busy[0].Index != 1 {
		t.Fatalf("Occupied reported %+v; only the card holding 12 GiB is above a 4 GiB threshold", busy)
	}
	if len(result.Occupied(16<<30)) != 0 {
		t.Error("a threshold above every card still named one")
	}
}

func TestSignallingPidOneIsRefusedAtTheLastMoment(t *testing.T) {
	// The listing already skips pid 1 and every ancestor. This proves the guard
	// below them, for a caller that reaches the signal directly: one guard for a
	// failure that restarts a container is one guard too few.
	if err := host.SignalGroupForTest(1); err == nil {
		t.Fatal("signalling pid 1 was allowed")
	}
	if err := host.SignalGroupForTest(0); err == nil {
		t.Fatal("signalling pid 0 was allowed; it is the caller's own process group")
	}
	if err := host.SignalGroupForTest(-1); err == nil {
		t.Fatal("signalling pid -1 was allowed; it means every process the user may signal")
	}
}

// Signals go to the process group, so the caller's group must remain protected.
func TestTheCleanupNeverSignalsItsOwnGroup(t *testing.T) {
	if err := host.SignalGroupForTest(os.Getpid()); err == nil {
		t.Fatal("the cleanup was allowed to signal its own group, and would stop half way through")
	}
}
