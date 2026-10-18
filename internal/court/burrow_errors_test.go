package court

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/lennrt/trial-lang/internal/docket"
)

// Fail a particular observation, after the case audit where applicable.
// Counting only calls through the source log leaves the replay copy untouched.
type burrowReadFailureLog struct {
	docket.Log
	topic     string
	attention bool
	failAt    int
	calls     int
	err       error
}

func (l *burrowReadFailureLog) ReadAll(ctx context.Context, topic string) ([]docket.Record, error) {
	if !l.attention && topic == l.topic {
		l.calls++
		if l.calls == l.failAt {
			return nil, l.err
		}
	}
	return l.Log.ReadAll(ctx, topic)
}

func (l *burrowReadFailureLog) Attention(ctx context.Context, c docket.Case) (docket.Attention, error) {
	if l.attention {
		l.calls++
		if l.calls == l.failAt {
			return docket.Attention{}, l.err
		}
	}
	return l.Log.Attention(ctx, c)
}

func TestBurrowPropagatesObservationFailures(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	log := docket.NewMemoryLog()
	t.Cleanup(log.Close)
	c, err := File(ctx, log, `FORM K-1. IN THE MATTER OF: survey-errors.
ARTICLE 1.
COMMIT "filed" TO THE ARCHIVE AS "document".
PROCLAIM "observed".
ADJOURN INDEFINITELY.`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := (&Court{Log: log, Case: c}).Proceed(ctx); err != nil {
		t.Fatal(err)
	}
	// A nonempty archive makes the survey inspect its catalog. This residual
	// record is unrelated to the replay and must remain an ordinary draft.
	if _, err := log.Append(ctx, c.Archive(), []byte("draft"), []byte(`{"t":"str","s":"unfinished"}`)); err != nil {
		t.Fatal(err)
	}
	baseline, err := SurveyBurrow(ctx, log)
	if err != nil || !baseline.Consistent() || len(baseline.Drafts[c.ID]) != 1 || baseline.Drafts[c.ID][0] != 1 {
		t.Fatalf("valid survey = %+v, %v; want one draft and a consistent audit", baseline, err)
	}

	stages := []struct {
		name      string
		topic     string
		attention bool
		failAt    int
	}{
		{name: "commencement ledger", topic: c.Ledger(), failAt: 1},
		{name: "archive", topic: c.Archive(), failAt: 1},
		{name: "catalog", topic: c.Catalog(), failAt: 1},
		// Audit reads source attention and records for its target and again
		// for final recovery. Fail the third read to reach the survey itself.
		{name: "attention", attention: true, failAt: 3},
		{name: "motion records", topic: c.Records(), failAt: 3},
	}
	for _, stage := range stages {
		t.Run(stage.name, func(t *testing.T) {
			for _, cause := range []error{errors.New("storage unavailable"), context.Canceled, docket.ErrTopicNotFound} {
				t.Run(cause.Error(), func(t *testing.T) {
					failing := &burrowReadFailureLog{
						Log: log, topic: stage.topic, attention: stage.attention,
						failAt: stage.failAt, err: cause,
					}
					report, err := SurveyBurrow(ctx, failing)
					if report != nil || !errors.Is(err, cause) {
						t.Fatalf("survey = %+v, %v; want no report and cause %v", report, err, cause)
					}
					if !strings.Contains(err.Error(), stage.name) {
						t.Fatalf("error %q does not identify failed %s read", err, stage.name)
					}
					if failing.calls != stage.failAt {
						t.Fatalf("continued after read failure: %d calls, want %d", failing.calls, stage.failAt)
					}
				})
			}
		})
	}
}
