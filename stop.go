package fleet

import (
	"context"
	"errors"
	"time"

	"github.com/arandu-io/framework/security"
)

// ErrUnquiesced means node reservations remain held because process termination
// or reservation release could not be confirmed.
var ErrUnquiesced = errors.New("fleet: termination and release are not confirmed")

// StopControl is the fenced cancellation and reservation boundary.
type StopControl interface {
	CancelNodes(context.Context, security.Subject, Job, []string) (Run, error)
	ReleaseNodes(Job, []string) error
}

// StopObservation reads measured states and may persist them in the application.
type StopObservation func(context.Context, []string) (map[string]NodeRun, error)

// StopAndRelease cancels a run, confirms quiescence on every selected node, and
// then releases its reservations. Unknown, missing, or unrelated terminal
// receipts cannot release a reservation. The caller owns the cleanup deadline
// and supplies a context independent of the cancelled execution.
func StopAndRelease(ctx context.Context, control StopControl, actor security.Subject, job Job, nodes []string, poll time.Duration, observe StopObservation) error {
	if ctx == nil || control == nil || observe == nil || job.ID == "" || job.Generation == 0 || poll <= 0 || len(nodes) == 0 {
		return errors.Join(ErrUnquiesced, errors.New("fleet: explicit stop identity, nodes, context and observer required"))
	}
	selected := append([]string(nil), nodes...)
	seen := make(map[string]bool, len(selected))
	for _, node := range selected {
		if node == "" || seen[node] {
			return errors.Join(ErrUnquiesced, errors.New("fleet: invalid stop node selection"))
		}
		seen[node] = true
	}
	if err := ctx.Err(); err != nil {
		return errors.Join(ErrUnquiesced, err)
	}
	answer, cancelErr := control.CancelNodes(ctx, actor, job, selected)
	if len(answer.Errors) != 0 {
		cancelErr = errors.Join(cancelErr, errors.New("fleet: cancellation failed on one or more nodes"))
	}
	for {
		states, statusErr := observe(ctx, selected)
		stopped := statusErr == nil
		for _, node := range selected {
			run, exists := states[node]
			if !exists || !stoppedRun(run, job) {
				stopped = false
			}
		}
		if stopped {
			if err := ctx.Err(); err != nil {
				return errors.Join(ErrUnquiesced, cancelErr, err)
			}
			if err := control.ReleaseNodes(job, selected); err != nil {
				return errors.Join(ErrUnquiesced, cancelErr, err)
			}
			// Cancellation is an intent, not proof of process state. A retry may
			// receive a conflict from an already idle/terminal worker. Independent
			// quiescence plus a successful fenced release completes that retry.
			return nil
		}
		timer := time.NewTimer(poll)
		select {
		case <-ctx.Done():
			timer.Stop()
			return errors.Join(ErrUnquiesced, cancelErr, statusErr, ctx.Err())
		case <-timer.C:
		}
	}
}

func stoppedRun(run NodeRun, job Job) bool {
	if run.State == StateIdle {
		return true
	}
	if run.ID != job.ID || run.Generation != job.Generation {
		return false
	}
	// A multi-stage application can share one fenced run across different node
	// actions. Each optional field, when declared, must still match exactly.
	if job.Action != "" && run.Action != job.Action ||
		job.ContractVersion != "" && run.ContractVersion != job.ContractVersion ||
		job.RuntimeDigest != "" && run.RuntimeDigest != job.RuntimeDigest ||
		job.ModelRecipe != "" && run.ModelRecipe != job.ModelRecipe ||
		job.ModelDigest != "" && run.ModelDigest != job.ModelDigest {
		return false
	}
	return run.State == StateSucceeded || run.State == StateFailed || run.State == StateCancelled
}
