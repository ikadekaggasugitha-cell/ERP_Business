package eventbus

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

// Kafka TLS is opt-in through the environment so local development can keep the
// plaintext listener while staging/production use the TLS listener (9093):
//
//	KAFKA_TLS_ENABLED=true
//	KAFKA_TLS_CA_FILE=/etc/kafka/ca/ca.crt   (Strimzi cluster CA; system roots when empty)
func tlsEnabled() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("KAFKA_TLS_ENABLED")), "true")
}

// loadTLSConfig returns nil when TLS is disabled. A misconfigured CA is an error:
// the caller must never fall back to plaintext once TLS was requested.
func loadTLSConfig() (*tls.Config, error) {
	if !tlsEnabled() {
		return nil, nil
	}
	cfg := &tls.Config{MinVersion: tls.VersionTLS12}
	caFile := strings.TrimSpace(os.Getenv("KAFKA_TLS_CA_FILE"))
	if caFile == "" {
		return cfg, nil
	}
	pem, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("read KAFKA_TLS_CA_FILE: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pem) {
		return nil, fmt.Errorf("KAFKA_TLS_CA_FILE %q contains no valid PEM certificate", caFile)
	}
	cfg.RootCAs = pool
	return cfg, nil
}

func mustTLSConfig() *tls.Config {
	cfg, err := loadTLSConfig()
	if err != nil {
		log.Fatalf("eventbus: kafka TLS misconfigured: %v", err)
	}
	return cfg
}

func newTransport() *kafka.Transport {
	return &kafka.Transport{TLS: mustTLSConfig(), DialTimeout: 5 * time.Second}
}

func newDialer() *kafka.Dialer {
	return &kafka.Dialer{TLS: mustTLSConfig(), Timeout: 10 * time.Second, DualStack: true}
}
