// Package eventbus is the SHARED Kafka layer (SDD BAB 6): the event Envelope
// (SDD-031), topic mapping, a Publisher (used by the outbox relay), and a
// Consumer helper with bounded retries + dead-letter queue (<topic>.dlq).
package eventbus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

// Envelope is the concrete event shape from SDD-031 (derived from outbox columns).
type Envelope struct {
	EventID        string          `json:"eventId"`
	EventType      string          `json:"eventType"`
	AggregateType  string          `json:"aggregateType"`
	AggregateID    string          `json:"aggregateId"`
	OccurredAt     time.Time       `json:"occurredAt"`
	CorrelationID  string          `json:"correlationId"`
	OrganizationID string          `json:"organizationId"`
	SchemaVersion  string          `json:"schemaVersion"`
	Data           json.RawMessage `json:"data"`
}

// ErrPoison indicates a deterministic error that should go straight to DLQ without retrying.
type ErrPoison struct{ Err error }

func (e ErrPoison) Error() string { return e.Err.Error() }
func (e ErrPoison) Unwrap() error { return e.Err }

// TopicFor maps a logical event_type ("gl.journal.posted") to the versioned
// topic name ("erp.gl.journal.posted.v1") per SAD 5.3.2 / SDD-032.
func TopicFor(eventType string) string { return "erp." + eventType + ".v1" }

// DLQTopic maps a topic to its dead-letter queue topic.
func DLQTopic(topic string) string { return topic + ".dlq" }

// CheckBrokers verifies TCP reachability to at least one configured broker (for /readyz).
func CheckBrokers(ctx context.Context, brokers []string) error {
	if len(brokers) == 0 {
		return nil
	}
	var lastErr error
	dialer := net.Dialer{Timeout: 2 * time.Second}
	for _, b := range brokers {
		conn, err := dialer.DialContext(ctx, "tcp", strings.TrimSpace(b))
		if err == nil {
			_ = conn.Close()
			return nil
		}
		lastErr = err
	}
	return fmt.Errorf("kafka broker unreachable: %w", lastErr)
}

// ---- Publisher (outbox relay) ----------------------------------------------

type Publisher struct{ w *kafka.Writer }

func NewPublisher(brokers []string) *Publisher {
	return &Publisher{w: &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Balancer:               &kafka.Hash{}, // partition by key (organization/aggregate)
		AllowAutoTopicCreation: true,
		RequiredAcks:           kafka.RequireAll, // acks=all (SDD 6.3)
		MaxAttempts:            5,
		BatchSize:              1,
	}}
}

// Publish sends one enveloped event. key preserves per-aggregate ordering.
func (p *Publisher) Publish(ctx context.Context, env Envelope) error {
	value, err := json.Marshal(env)
	if err != nil {
		return err
	}
	return p.w.WriteMessages(ctx, kafka.Message{
		Topic: TopicFor(env.EventType),
		Key:   []byte(env.AggregateID),
		Value: value,
		Headers: []kafka.Header{
			{Key: "organization_id", Value: []byte(env.OrganizationID)},
			{Key: "correlation_id", Value: []byte(env.CorrelationID)},
			{Key: "event_type", Value: []byte(env.EventType)},
		},
	})
}

func (p *Publisher) Close() error {
	if p == nil || p.w == nil {
		return nil
	}
	return p.w.Close()
}

// ---- Consumer --------------------------------------------------------------

type DLQRecord struct {
	OriginalTopic   string          `json:"originalTopic"`
	Partition       int             `json:"partition"`
	Offset          int64           `json:"offset"`
	Key             string          `json:"key"`
	FailedAt        time.Time       `json:"failedAt"`
	Attempts        int             `json:"attempts"`
	Error           string          `json:"error"`
	OriginalPayload json.RawMessage `json:"originalPayload"`
}

type Consumer struct {
	r   *kafka.Reader
	dlq *kafka.Writer
}

func NewConsumer(brokers []string, groupID string, topics []string) *Consumer {
	return &Consumer{
		r: kafka.NewReader(kafka.ReaderConfig{
			Brokers:     brokers,
			GroupID:     groupID,
			GroupTopics: topics,
			MinBytes:    1,
			MaxBytes:    10e6,
		}),
		dlq: &kafka.Writer{
			Addr:                   kafka.TCP(brokers...),
			Balancer:               &kafka.Hash{},
			AllowAutoTopicCreation: true,
			RequiredAcks:           kafka.RequireAll,
		},
	}
}

// maxAttempts bounds per-message handler retries before routing to <topic>.dlq.
const maxAttempts = 5

// Run reads messages until ctx is cancelled. Offsets commit ONLY after handle succeeds
// OR after routing the poison/exhausted message cleanly to <topic>.dlq.
func (c *Consumer) Run(ctx context.Context, handle func(context.Context, Envelope) error) error {
	for {
		m, err := c.r.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		var env Envelope
		if err := json.Unmarshal(m.Value, &env); err != nil {
			log.Printf("[consumer] malformed message topic=%s offset=%d: %v (routing to DLQ)", m.Topic, m.Offset, err)
			if derr := c.publishDLQ(ctx, m, 1, fmt.Errorf("malformed json: %w", err)); derr != nil {
				log.Printf("[consumer] dlq write failed topic=%s offset=%d: %v (retrying)", m.Topic, m.Offset, derr)
				continue
			}
			c.commit(ctx, m)
			continue
		}
		var herr error
		attempts := 0
		for attempt := 1; attempt <= maxAttempts; attempt++ {
			attempts = attempt
			if herr = handle(ctx, env); herr == nil {
				break
			}
			var poison ErrPoison
			var poisonPtr *ErrPoison
			if errors.As(herr, &poison) || errors.As(herr, &poisonPtr) {
				break
			}
			log.Printf("[consumer] handle failed (attempt %d/%d) event=%s: %v", attempt, maxAttempts, env.EventID, herr)
			if attempt < maxAttempts {
				select {
				case <-ctx.Done():
					return nil
				case <-time.After(backoff(attempt)):
				}
			}
		}
		if herr != nil {
			log.Printf("[consumer] routing event=%s topic=%s to %s after %d attempts: %v", env.EventID, m.Topic, DLQTopic(m.Topic), attempts, herr)
			if derr := c.publishDLQ(ctx, m, attempts, herr); derr != nil {
				// Do not commit offset when DLQ itself failed; let Kafka redeliver after recovery.
				log.Printf("[consumer] dlq write failed event=%s: %v (not committing offset)", env.EventID, derr)
				continue
			}
		}
		c.commit(ctx, m)
	}
}

func (c *Consumer) publishDLQ(ctx context.Context, m kafka.Message, attempts int, cause error) error {
	if c.dlq == nil {
		return nil
	}
	payload := m.Value
	if !json.Valid(payload) {
		raw, _ := json.Marshal(string(payload))
		payload = raw
	}
	rec := DLQRecord{
		OriginalTopic:   m.Topic,
		Partition:       m.Partition,
		Offset:          m.Offset,
		Key:             string(m.Key),
		FailedAt:        time.Now().UTC(),
		Attempts:        attempts,
		Error:           cause.Error(),
		OriginalPayload: payload,
	}
	val, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	return c.dlq.WriteMessages(ctx, kafka.Message{
		Topic: DLQTopic(m.Topic),
		Key:   m.Key,
		Value: val,
		Headers: append(m.Headers,
			kafka.Header{Key: "dlq_error", Value: []byte(cause.Error())},
			kafka.Header{Key: "dlq_original_topic", Value: []byte(m.Topic)},
		),
	})
}

func (c *Consumer) commit(ctx context.Context, m kafka.Message) {
	if err := c.r.CommitMessages(ctx, m); err != nil && ctx.Err() == nil {
		log.Printf("[consumer] commit failed offset=%d: %v", m.Offset, err)
	}
}

// backoff grows quadratically, capped at 5s: ~0.2, 0.8, 1.8, 3.2, 5s.
func backoff(attempt int) time.Duration {
	d := time.Duration(attempt*attempt) * 200 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}

func (c *Consumer) Close() error {
	var firstErr error
	if c.r != nil {
		firstErr = c.r.Close()
	}
	if c.dlq != nil {
		if err := c.dlq.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
