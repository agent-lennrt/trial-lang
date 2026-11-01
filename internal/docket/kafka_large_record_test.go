package docket

import (
	"bytes"
	"context"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestE2E_LargeRecords(t *testing.T) {
	broker := os.Getenv("TRIAL_E2E_BROKER")
	if broker == "" {
		t.Skip("TRIAL_E2E_BROKER is not set")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	defer cancel()
	log, err := OpenKafkaLog(ctx, broker)
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	c, err := NewCase()
	if err != nil {
		t.Fatal(err)
	}
	// Use a case topic so execution, batch, and ordinary producers all write
	// the identical payload under the identical explicitly raised broker limit.
	topic := c.Proclamations()
	responses, err := log.adm.CreateTopics(ctx, 1, 1, map[string]*string{
		"max.message.bytes": new(strconv.Itoa(maxKafkaRecordBatchBytes)),
	}, topic, c.AttentionTopic(), c.Proceedings(), c.Summons())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if _, err := log.adm.DeleteTopics(cleanup, topic, c.AttentionTopic(), c.Proceedings(), c.Summons()); err != nil {
			t.Errorf("delete large-record topics: %v", err)
		}
	}()
	for _, response := range responses {
		if response.Err != nil {
			t.Fatal(response.Err)
		}
	}
	value := bytes.Repeat([]byte("large record payload "), (2<<20)/21+1)
	if _, err := log.Append(ctx, topic, []byte("append"), value); err != nil {
		t.Fatalf("Append: %v", err)
	}
	if _, err := log.AppendBatch(ctx, []StepAppend{{Topic: topic, Key: []byte("batch"), Value: value}}); err != nil {
		t.Fatalf("AppendBatch: %v", err)
	}
	// The execution producer must honor the same limit as paperwork producers.
	if err := log.Commit(ctx, c, Step{Appends: []StepAppend{{Topic: topic, Key: []byte("commit"), Value: value}}}); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	records, err := log.ReadAll(ctx, topic)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 3 {
		t.Fatalf("read %d records, want 3", len(records))
	}
	for i, key := range []string{"append", "batch", "commit"} {
		if string(records[i].Key) != key || !bytes.Equal(records[i].Value, value) {
			t.Fatalf("record %d did not roundtrip", i)
		}
	}
	att, err := log.Attention(ctx, c)
	if err != nil || !att.Started {
		t.Fatalf("committed attention: %+v, %v", att, err)
	}
}
