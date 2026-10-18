package court

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/lennrt/trial-lang/internal/docket"
)

// Any method other than the two dependency reads panics through the nil Log.
// This makes case creation, statute publication, and execution fail the test.
type checkReadOnlyLog struct {
	docket.Log
	source docket.Log
}

func (log checkReadOnlyLog) ListStatutes(ctx context.Context) ([]string, error) {
	return log.source.ListStatutes(ctx)
}

func (log checkReadOnlyLog) ReadAll(ctx context.Context, topic string) ([]docket.Record, error) {
	return log.source.ReadAll(ctx, topic)
}

type checkCancelledReadLog struct {
	docket.Log
	cancel context.CancelFunc
}

func (log checkCancelledReadLog) ListStatutes(ctx context.Context) ([]string, error) {
	log.cancel()
	return nil, ctx.Err()
}

func TestCheckDoesNotWriteOrExecute(t *testing.T) {
	mem := docket.NewMemoryLog()
	defer mem.Close()
	if _, _, err := Enact(t.Context(), mem, "FORM S-1.\nIN THE MATTER OF: dependency.\nTHE OFFICE OF helper, CONCERNING n. REMAND WITH n."); err != nil {
		t.Fatal(err)
	}
	readOnly := checkReadOnlyLog{source: mem}
	for _, source := range []string{
		"FORM K-1.\nIN THE MATTER OF: loop.\nARTICLE 1. REFER TO ARTICLE 1.",
		"FORM K-1.\nIN THE MATTER OF: input.\nARTICLE 1. AWAIT SUMMONS, FILED UNDER value.",
		"FORM K-1.\nIN THE MATTER OF: caller.\nINCORPORATE BY REFERENCE dependency.\nARTICLE 1. PROCLAIM THE FINDING OF helper REGARDING 1.",
		"FORM S-1.\nIN THE MATTER OF: draft.\nINCORPORATE BY REFERENCE dependency.\nTHE OFFICE OF other. REMAND WITH THE FINDING OF helper REGARDING 1.",
	} {
		for range 2 {
			if err := Check(t.Context(), readOnly, source); err != nil {
				t.Fatalf("read-only validation failed: %v", err)
			}
		}
	}
	err := Check(t.Context(), readOnly, "FORM K-1.\nIN THE MATTER OF: caller.\nINCORPORATE BY REFERENCE draft.\nARTICLE 1. ADJOURN INDEFINITELY.")
	assertSourceRejection(t, err, "", 3, "no statute named")
	err = Check(t.Context(), readOnly, "FORM K-1.\nIN THE MATTER OF: collision.\nINCORPORATE BY REFERENCE dependency.\nARTICLE 1. ADJOURN INDEFINITELY.\nTHE OFFICE OF helper. REMAND.")
	assertSourceRejection(t, err, "dependency", 3, "established more than once")
}

func TestCheckRejectsInvalidOrCancelledSource(t *testing.T) {
	for _, source := range []string{
		"FORM K-9.",
		"FORM K-1.\nIN THE MATTER OF: root.\nARTICLE 1. REFER TO ARTICLE 99.",
	} {
		if err := Check(t.Context(), checkReadOnlyLog{}, source); err == nil {
			t.Fatalf("invalid source accepted: %q", source)
		}
	}
	if err := Check(t.Context(), checkReadOnlyLog{}, "FORM K-2.\nIN THE MATTER OF: root.\nARTICLE 2. ADJOURN INDEFINITELY."); err == nil || !strings.Contains(err.Error(), "K-1 case or Form S-1 statute") {
		t.Fatalf("unexpected supplemental-form result: %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := Check(ctx, checkReadOnlyLog{}, "FORM K-1.\nIN THE MATTER OF: root.\nARTICLE 1. ADJOURN INDEFINITELY."); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled validation returned %v", err)
	}
	ctx, cancel = context.WithCancel(t.Context())
	defer cancel()
	err := Check(ctx, checkCancelledReadLog{cancel: cancel}, "FORM K-1.\nIN THE MATTER OF: root.\nINCORPORATE BY REFERENCE dependency.\nARTICLE 1. ADJOURN INDEFINITELY.")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation during dependency lookup became a source diagnostic: %v", err)
	}
}
