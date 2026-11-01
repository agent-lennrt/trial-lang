package court

import (
	"context"
	"math"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/lennrt/trial-lang/internal/docket"
	"github.com/lennrt/trial-lang/internal/law"
)

func TestRecoverRejectsMissingPushValue(t *testing.T) {
	for _, payload := range []string{`{"op":"PUSH"}`, `{"op":"PUSH","value":null}`} {
		t.Run(payload, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			log, c := convene(t, "FORM K-1.\nIN THE MATTER OF: damaged-history.\nARTICLE 1.\nADJOURN INDEFINITELY.")
			defer log.Close()
			if _, err := log.Append(ctx, c.Dossier(), nil, []byte(payload)); err != nil {
				t.Fatal(err)
			}
			if err := (&Court{Log: log, Case: c}).Recover(ctx); err == nil || !strings.Contains(err.Error(), "PUSH") {
				t.Fatalf("Recover error = %v; want missing PUSH value", err)
			}
		})
	}
}

func TestProceedRejectsInvalidStoredArity(t *testing.T) {
	for _, op := range []string{law.OpSchedule, law.OpPetitionUnder} {
		for _, count := range []int{-1, math.MaxInt} {
			t.Run(op+"/"+strconv.Itoa(count), func(t *testing.T) {
				ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
				defer cancel()
				log, c := convene(t, "FORM K-1.\nIN THE MATTER OF: damaged-code.\nARTICLE 1.")
				defer log.Close()
				if _, err := log.Append(ctx, c.Proceedings(), nil, (law.Instr{Op: op, Count: count}).Marshal()); err != nil {
					t.Fatal(err)
				}
				out, err := (&Court{Log: log, Case: c}).Proceed(ctx)
				if err != nil || out != OutcomeGuilty {
					t.Fatalf("Proceed = %v, %v; want verdict", out, err)
				}
				recs, err := log.ReadAll(ctx, c.Verdicts())
				if err != nil || len(recs) != 1 {
					t.Fatalf("verdicts=%v, %v", recs, err)
				}
				if count > 0 && !strings.Contains(string(recs[0].Value), "file you requested") {
					t.Fatalf("want established dossier underflow: %s", recs[0].Value)
				}
			})
		}
	}
}

func TestRecoverRejectsNegativeCursors(t *testing.T) {
	for _, field := range []string{"PC", "Summons", "Ledger", "Gazette"} {
		t.Run(field, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			log, c := convene(t, "FORM K-1.\nIN THE MATTER OF: damaged-attention.\nARTICLE 1.\nPROCLAIM THE DISCRETION OF THE COURT BETWEEN 1 AND 1.")
			defer log.Close()
			step := docket.Step{}
			switch field {
			case "PC":
				step.PC = -1
			case "Summons":
				step.Summons = -1
			case "Ledger":
				step.Ledger = -1
			case "Gazette":
				step.Gazette = -1
			}
			if err := log.Commit(ctx, c, step); err != nil {
				t.Fatal(err)
			}
			if err := (&Court{Log: log, Case: c}).Recover(ctx); err == nil || !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(field)) {
				t.Fatalf("Recover error = %v; want invalid %s", err, field)
			}
		})
	}
}

func TestProceedAcceptsZeroStoredArity(t *testing.T) {
	for _, test := range []struct {
		name         string
		instructions []law.Instr
	}{
		{"schedule", []law.Instr{{Op: law.OpSchedule}, {Op: law.OpProclaim}, {Op: law.OpAdjourn}}},
		{"petition", []law.Instr{{Op: law.OpPower, Name: "empty-office", Target: 3}, {Op: law.OpPetitionUnder}, {Op: law.OpAdjourn}, {Op: law.OpRemand}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			log, c := convene(t, "FORM K-1.\nIN THE MATTER OF: empty-arguments.\nARTICLE 1.")
			defer log.Close()
			for _, in := range test.instructions {
				if _, err := log.Append(ctx, c.Proceedings(), nil, in.Marshal()); err != nil {
					t.Fatal(err)
				}
			}
			out, err := (&Court{Log: log, Case: c}).Proceed(ctx)
			if err != nil || out != OutcomeAdjourned {
				t.Fatalf("Proceed = %v, %v; want adjournment", out, err)
			}
			ct := &Court{Log: log, Case: c}
			if err := ct.Recover(ctx); err != nil {
				t.Fatal(err)
			}
			if len(ct.stack) != 0 || len(ct.frames) != 0 {
				t.Fatalf("recovered stack=%v, frames=%v", ct.stack, ct.frames)
			}
			if test.name == "schedule" && len(proclamations(t, log, c)) != 1 {
				t.Fatal("empty schedule was not proclaimed")
			}
		})
	}
}
