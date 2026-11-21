// Command dlq-replay inspects and replays dead-lettered Kafka messages (<topic>.dlq)
// back to their original domain topic (<topic>) with audit headers.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/sinergiinvestasiproperti/erp/services/shared/eventbus"
)

func main() {
	var (
		brokersFlag = flag.String("brokers", os.Getenv("KAFKA_BROKERS"), "comma-separated Kafka brokers (default: $KAFKA_BROKERS or localhost:19092)")
		topicFlag   = flag.String("topic", "", "DLQ topic name (e.g. erp.gl.journal.posted.v1.dlq) or base topic (erp.gl.journal.posted.v1)")
		maxMessages = flag.Int("limit", 100, "maximum number of DLQ messages to inspect/replay")
		dryRun      = flag.Bool("dry-run", false, "inspect DLQ records without republishing")
		timeout     = flag.Duration("timeout", 10*time.Second, "read/write timeout")
	)
	flag.Parse()

	brokersRaw := strings.TrimSpace(*brokersFlag)
	if brokersRaw == "" {
		brokersRaw = "localhost:19092"
	}
	var brokers []string
	for _, b := range strings.Split(brokersRaw, ",") {
		if b = strings.TrimSpace(b); b != "" {
			brokers = append(brokers, b)
		}
	}
	topic := strings.TrimSpace(*topicFlag)
	if topic == "" {
		log.Fatal("-topic is required (e.g. -topic erp.gl.journal.posted.v1.dlq)")
	}
	dlqTopic := topic
	if !strings.HasSuffix(dlqTopic, ".dlq") {
		dlqTopic = eventbus.DLQTopic(dlqTopic)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   brokers,
		Topic:     dlqTopic,
		Partition: 0,
		MinBytes:  1,
		MaxBytes:  10e6,
	})
	defer reader.Close()

	writer := &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Balancer:               &kafka.Hash{},
		AllowAutoTopicCreation: true,
		RequiredAcks:           kafka.RequireAll,
	}
	defer writer.Close()

	replayed := 0
	for i := 0; i < *maxMessages; i++ {
		readCtx, readCancel := context.WithTimeout(ctx, 2*time.Second)
		msg, err := reader.ReadMessage(readCtx)
		readCancel()
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				break
			}
			log.Fatalf("read dlq message: %v", err)
		}
		var rec eventbus.DLQRecord
		if err := json.Unmarshal(msg.Value, &rec); err != nil {
			log.Printf("skip offset=%d: invalid DLQRecord JSON: %v", msg.Offset, err)
			continue
		}
		targetTopic := strings.TrimSpace(rec.OriginalTopic)
		if targetTopic == "" {
			targetTopic = strings.TrimSuffix(dlqTopic, ".dlq")
		}
		fmt.Printf("[dlq] offset=%d origTopic=%s key=%s attempts=%d failedAt=%s err=%q\n",
			msg.Offset, targetTopic, rec.Key, rec.Attempts, rec.FailedAt.Format(time.RFC3339), rec.Error)
		if *dryRun {
			continue
		}
		err = writer.WriteMessages(ctx, kafka.Message{
			Topic: targetTopic,
			Key:   []byte(rec.Key),
			Value: rec.OriginalPayload,
			Headers: []kafka.Header{
				{Key: "dlq_replayed_from", Value: []byte(dlqTopic)},
				{Key: "dlq_replayed_at", Value: []byte(time.Now().UTC().Format(time.RFC3339))},
			},
		})
		if err != nil {
			log.Fatalf("republish offset=%d to %s failed: %v", msg.Offset, targetTopic, err)
		}
		replayed++
	}
	fmt.Printf("done: replayed=%d dryRun=%v dlqTopic=%s\n", replayed, *dryRun, dlqTopic)
}
