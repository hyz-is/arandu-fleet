package host

import (
	"math"
	"testing"
)

func TestMatchedProgramRequiresExecutableOrExplicitScriptPosition(t *testing.T) {
	programs := map[string]bool{"owned-worker": true, "owned-script": true}
	interpreters := map[string]bool{"script-runner": true}
	for _, test := range []struct {
		name string
		args []string
		want string
	}{
		{"executable", []string{"/opt/tools/owned-worker", "--job", "fixture"}, "owned-worker"},
		{"script", []string{"/usr/bin/script-runner", "/opt/tools/owned-script", "fixture"}, "owned-script"},
		{"unrelated argument", []string{"editor", "/opt/tools/owned-script"}, ""},
		{"later argument", []string{"script-runner", "other-script", "owned-script"}, ""},
		{"command expression", []string{"script-runner", "-c", "owned-script"}, ""},
		{"module option", []string{"script-runner", "-m", "owned-script"}, ""},
		{"empty script", []string{"script-runner", ""}, ""},
		{"missing script", []string{"script-runner"}, ""},
		{"empty argv", nil, ""},
		{"empty executable", []string{"", "owned-script"}, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := matchedProgram(test.args, programs, interpreters); got != test.want {
				t.Fatalf("program = %q, want %q", got, test.want)
			}
		})
	}
	if got := matchedProgram([]string{"script-runner", "owned-script"}, programs, nil); got != "" {
		t.Fatalf("matched script without an explicit interpreter: %q", got)
	}
}

func TestCleanRejectsInvalidFractionBeforeHostAccess(t *testing.T) {
	for _, fraction := range []float64{-1, 2, math.NaN(), math.Inf(1)} {
		result, err := (Housekeeping{VRAMFraction: fraction}).Clean(t.Context(), true, true)
		if err == nil || err.Error() != "host: the VRAM fraction must be zero or in (0, 1]" || len(result.Signalled) != 0 || len(result.Removed) != 0 {
			t.Fatalf("invalid fraction %v was not rejected before cleanup: %+v, %v", fraction, result, err)
		}
	}
}
