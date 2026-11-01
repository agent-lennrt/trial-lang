package docket

import (
	"testing"

	"github.com/twmb/franz-go/pkg/kgo"
)

func TestKafkaProducersAdmitMaximumRecord(t *testing.T) {
	log := &KafkaLog{brokers: []string{"localhost:1"}}
	clients := map[string]func() (*kgo.Client, error){
		"paperwork": func() (*kgo.Client, error) { return newKafkaProducerClient(log.brokers) },
		"execution": func() (*kgo.Client, error) {
			return log.newTransactionClient(Case{ID: "case-000000000000000000000001"})
		},
	}
	for name, create := range clients {
		t.Run(name, func(t *testing.T) {
			client, err := create()
			if err != nil {
				t.Fatal(err)
			}
			defer client.Close()
			limit := client.OptValue(kgo.ProducerBatchMaxBytes).(int32)
			// A v2 batch has a 61-byte header; each key and value length needs at
			// most five varint bytes. Allow extra room for the record's other fields.
			if limit < 2*MaxRecordBytes+128 {
				t.Fatalf("producer batch limit %d cannot encode maximum key and value", limit)
			}
			if limit > 2*MaxRecordBytes+1024 {
				t.Fatalf("producer batch limit %d is unnecessarily large", limit)
			}
		})
	}
}
