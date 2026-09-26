package unit_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/arandu-io/framework/security"
	fleet "github.com/tayi-ai/arandu-fleet"
)

type stopControl struct {
	cancelled, released int
	releaseError        error
	cancelError         error
}

func (c *stopControl) CancelNodes(context.Context, security.Subject, fleet.Job, []string) (fleet.Run, error) {
	c.cancelled++
	return fleet.Run{}, c.cancelError
}
func (c *stopControl) ReleaseNodes(fleet.Job, []string) error { c.released++; return c.releaseError }

func TestStopAndReleaseRequiresMeasuredQuiescence(t *testing.T) {
	job := fleet.Job{ID: "job", Generation: 2, Action: "train", RuntimeDigest: "runtime"}
	terminal := fleet.NodeRun{ID: job.ID, Generation: job.Generation, Action: job.Action, RuntimeDigest: job.RuntimeDigest, State: fleet.StateCancelled}
	cases := []struct {
		name              string
		run               fleet.NodeRun
		missing, accepted bool
	}{
		{name: "terminal", run: terminal, accepted: true},
		{name: "idle", run: fleet.NodeRun{State: fleet.StateIdle}, accepted: true},
		{name: "unknown", run: fleet.NodeRun{ID: job.ID, Generation: 2, State: fleet.StateUnknown}},
		{name: "other run", run: fleet.NodeRun{ID: "other", Generation: 2, State: fleet.StateSucceeded}},
		{name: "old generation", run: fleet.NodeRun{ID: job.ID, Generation: 1, State: fleet.StateCancelled}},
		{name: "wrong runtime", run: fleet.NodeRun{ID: job.ID, Generation: 2, Action: "train", RuntimeDigest: "other", State: fleet.StateSucceeded}},
		{name: "missing", missing: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			control := &stopControl{}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			observe := func(context.Context, []string) (map[string]fleet.NodeRun, error) {
				if !tc.accepted {
					cancel()
				}
				if tc.missing {
					return map[string]fleet.NodeRun{}, nil
				}
				return map[string]fleet.NodeRun{"node": tc.run}, nil
			}
			err := fleet.StopAndRelease(ctx, control, security.Subject{}, job, []string{"node"}, time.Millisecond, observe)
			if tc.accepted && (err != nil || control.released != 1) {
				t.Fatalf("confirmed stop failed: %v releases=%d", err, control.released)
			}
			if !tc.accepted && (!errors.Is(err, fleet.ErrUnquiesced) || control.released != 0) {
				t.Fatalf("unconfirmed stop released: %v releases=%d", err, control.released)
			}
		})
	}
}

func TestStopAndReleaseWaitsForEveryNodeAndPropagatesReleaseFailure(t *testing.T) {
	failure := errors.New("reservation failure")
	control := &stopControl{releaseError: failure}
	job := fleet.Job{ID: "job", Generation: 1}
	calls := 0
	observe := func(context.Context, []string) (map[string]fleet.NodeRun, error) {
		calls++
		states := map[string]fleet.NodeRun{"one": {State: fleet.StateIdle}}
		if calls > 1 {
			states["two"] = fleet.NodeRun{State: fleet.StateIdle}
		}
		return states, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := fleet.StopAndRelease(ctx, control, security.Subject{}, job, []string{"one", "two"}, time.Millisecond, observe)
	if !errors.Is(err, failure) || !errors.Is(err, fleet.ErrUnquiesced) || control.released != 1 || calls != 2 {
		t.Fatalf("release failure lost: %v calls=%d releases=%d", err, calls, control.released)
	}
}

func TestStopAndReleaseReconcilesAlreadyStoppedCancellationConflict(t *testing.T) {
	control := &stopControl{cancelError: errors.New("cancel: already idle")}
	observe := func(context.Context, []string) (map[string]fleet.NodeRun, error) {
		return map[string]fleet.NodeRun{"node": {State: fleet.StateIdle}}, nil
	}
	err := fleet.StopAndRelease(context.Background(), control, security.Subject{}, fleet.Job{ID: "job", Generation: 1}, []string{"node"}, time.Millisecond, observe)
	if err != nil || control.released != 1 {
		t.Fatalf("already stopped reconciliation failed: %v", err)
	}
}
