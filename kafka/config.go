package kafka

import (
	"crypto/tls"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"
)

// Config holds Kafka connection settings including SASL auth.
type Config struct {
	Brokers   []string
	Username  string
	Password  string
	Protocol  string // SaslSsl, SaslPlaintext, or empty for plaintext
	Mechanism string // ScramSha512, ScramSha256, Plain
}

// NewDialer creates a kafka.Dialer with optional SASL/TLS based on config.
func (c *Config) NewDialer() *kafka.Dialer {
	dialer := &kafka.Dialer{
		Timeout:   10 * time.Second,
		DualStack: true,
	}

	// SASL mechanism
	mechanism := c.buildSASLMechanism()
	if mechanism != nil {
		dialer.SASLMechanism = mechanism
	}

	// TLS
	if c.isTLS() {
		dialer.TLS = &tls.Config{}
	}

	return dialer
}

// Transport creates a kafka.Transport with optional SASL/TLS for the Writer.
func (c *Config) Transport() *kafka.Transport {
	t := &kafka.Transport{}

	mechanism := c.buildSASLMechanism()
	if mechanism != nil {
		t.SASL = mechanism
	}

	if c.isTLS() {
		t.TLS = &tls.Config{}
	}

	return t
}

func (c *Config) buildSASLMechanism() sasl.Mechanism {
	if c.Protocol == "" || strings.EqualFold(c.Protocol, "PLAINTEXT") {
		return nil
	}
	if c.Username == "" || c.Password == "" {
		return nil
	}

	switch strings.ToLower(c.Mechanism) {
	case "scramsha512":
		m, err := scram.Mechanism(scram.SHA512, c.Username, c.Password)
		if err != nil {
			return nil
		}
		return m
	case "scramsha256":
		m, err := scram.Mechanism(scram.SHA256, c.Username, c.Password)
		if err != nil {
			return nil
		}
		return m
	case "plain":
		return &plain.Mechanism{Username: c.Username, Password: c.Password}
	default:
		m, err := scram.Mechanism(scram.SHA512, c.Username, c.Password)
		if err != nil {
			return nil
		}
		return m
	}
}

func (c *Config) isTLS() bool {
	return strings.EqualFold(c.Protocol, "SaslSsl")
}
