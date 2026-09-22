package unit_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	fleet "github.com/tayi-ai/arandu-fleet"
)

func TestPartialDispatchPreservesOriginalNodeReceipt(t *testing.T) {
	worker := newFake()
	worker.refuse["one"] = true
	cp, err := fleet.NewControlPlane(tayiDeclared(), inventory(), worker)
	if err != nil {
		t.Fatal(err)
	}
	run, err := cp.Dispatch(context.Background(), operator(), fleet.ActionDiagnostics, "train")
	if err == nil {
		t.Fatal("expected a partial dispatch failure")
	}
	want := []string{"one", "three", "five"}
	if !slices.Equal(run.Nodes, want) {
		t.Fatalf("dispatch receipt changed after reconciliation: got %v, want %v", run.Nodes, want)
	}
	active, ok := cp.InFlight()
	if !ok || !slices.Equal(active.Nodes, []string{"three", "five"}) {
		t.Fatalf("unconfirmed nodes were not retained: %+v", active)
	}
}

func TestTerminalReleaseDoesNotMutatePublishedReceipt(t *testing.T) {
	cp, err := fleet.NewControlPlane(tayiDeclared(), inventory(), newFake())
	if err != nil {
		t.Fatal(err)
	}
	job := fleet.Job{ID: "receipt-isolation", Action: fleet.ActionDiagnostics, Generation: 1}
	run, err := cp.DispatchNodes(context.Background(), operator(), job, []string{"one", "three", "five"})
	if err != nil {
		t.Fatal(err)
	}
	if err := cp.ReleaseNodes(job, []string{"one"}); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(run.Nodes, []string{"one", "three", "five"}) {
		t.Fatalf("terminal reconciliation rewrote the returned receipt: %v", run.Nodes)
	}
}

func TestCallerCannotMutateTheActiveReservation(t *testing.T) {
	cp, err := fleet.NewControlPlane(tayiDeclared(), inventory(), newFake())
	if err != nil {
		t.Fatal(err)
	}
	run, err := cp.Dispatch(context.Background(), operator(), fleet.ActionDiagnostics, "train")
	if err != nil {
		t.Fatal(err)
	}
	run.Nodes[0] = "twentyone"
	active, ok := cp.InFlight()
	if !ok || !slices.Equal(active.Nodes, []string{"one", "three", "five"}) {
		t.Fatalf("caller changed active reservation: %+v", active)
	}
}

type dispatchRoundTrip func(*http.Request) (*http.Response, error)

func (f dispatchRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func delayedWorker(t *testing.T) *fleet.HTTPWorker {
	t.Helper()
	w, err := fleet.NewHTTPWorker("test-token", func(fleet.Node) int { return 8787 })
	if err != nil {
		t.Fatal(err)
	}
	w.Client.Timeout = 10 * time.Millisecond
	w.Client.Transport = dispatchRoundTrip(func(r *http.Request) (*http.Response, error) {
		timer := time.NewTimer(50 * time.Millisecond)
		defer timer.Stop()
		select {
		case <-r.Context().Done():
			return nil, r.Context().Err()
		case <-timer.C:
			return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"accepted":true}`)), Request: r}, nil
		}
	})
	return w
}

func TestSubmissionBudgetAllowsSlowAdmissionWithoutSlowingStatus(t *testing.T) {
	w := delayedWorker(t)
	w.SubmitTimeout = time.Second
	n := fleet.Node{ID: "one", IP: "127.0.0.1"}
	job := fleet.Job{ID: "slow-admission", Action: fleet.ActionDiagnostics}
	if _, err := w.Submit(context.Background(), n, job); err != nil {
		t.Fatalf("slow admission refused: %v", err)
	}
	if _, err := w.Status(context.Background(), n); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("status lost its short timeout: %v", err)
	}
	if _, err := w.Cancel(context.Background(), n, job); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("cancel lost its short timeout: %v", err)
	}
	if w.Client.Timeout != 10*time.Millisecond {
		t.Fatal("shared HTTP client was mutated")
	}
}

func TestSubmissionBudgetHonorsCallerDeadlineAndZeroDefault(t *testing.T) {
	w := delayedWorker(t)
	n := fleet.Node{ID: "one", IP: "127.0.0.1"}
	job := fleet.Job{ID: "bounded-admission", Action: fleet.ActionDiagnostics}
	if _, err := w.Submit(context.Background(), n, job); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("zero did not preserve client timeout: %v", err)
	}
	w.SubmitTimeout = time.Second
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if _, err := w.Submit(ctx, n, job); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("caller deadline was ignored: %v", err)
	}
	w.SubmitTimeout = -time.Second
	if _, err := w.Submit(context.Background(), n, job); err == nil {
		t.Fatal("negative submission budget accepted")
	}
}
