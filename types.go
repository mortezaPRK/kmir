package main

import (
	"time"

	"github.com/twmb/go-franz/pkg/kgo"
	"github.com/twmb/go-franz/pkg/kversion"
)

// Sasl defines the SASL configuration for a broker.
type Sasl struct {
	Enabled   bool   `long:"enabled" env:"ENABLED" description:"Enable SASL"`
	Username  string `long:"username" env:"USERNAME" description:"SASL username"`
	Password  string `long:"password" env:"PASSWORD" description:"SASL password"`
	Mechanism string `long:"mechanism" env:"MECHANISM" description:"SASL mechanism"`
}

// Tls defines the TLS configuration for a broker.
type Tls struct {
	Enabled  bool   `long:"enabled" env:"ENABLED" description:"Enable TLS"`
	Cert     string `long:"cert" env:"CERT" description:"TLS certificate"`
	Insecure bool   `long:"insecure" env:"INSECURE" description:"Skip TLS verification"`
}

// BrokerOptions defines the configuration for a Kafka broker.
type BrokerOptions struct {
	Brokers []string      `long:"brokers" env:"BROKERS" env-delim:"," description:"Comma-separated list of Kafka brokers" required:"true"`
	Tls     Tls           `group:"TLS" namespace:"tls" env-namespace:"TLS"`
	Sasl    Sasl          `group:"SASL" namespace:"sasl" env-namespace:"SASL"`
	Timeout time.Duration `long:"timeout" env:"TIMEOUT" description:"Timeout for Kafka" default:"10s"`
}

// TopicOption defines the configuration for a topic.
type TopicOption struct {
	Offset             int64
	PerPartitionOffset map[int32]int64
}

func (to TopicOption) OffsetOf(partition int32) (int64, bool) {
	if to.PerPartitionOffset == nil {
		return to.Offset, true
	}

	offset, ok := to.PerPartitionOffset[partition]
	return offset, ok
}

// Options defines the command line options for the application.
type Options struct {
	Source       BrokerOptions `group:"Source" namespace:"source" env-namespace:"SOURCE"`
	Sink         BrokerOptions `group:"Sink" namespace:"sink" env-namespace:"SINK"`
	ClientID     string        `long:"client-id" env:"CLIENT_ID" description:"Client ID" required:"true"`
	KafkaVersion string        `long:"kafka-version" env:"KAFKA_VERSION" description:"Kafka version" required:"true"`
}

// Config defines the configuration for the whole application.
type Config struct {
	Sink         []kgo.Opt
	Source       []kgo.Opt
	ClientID     string
	KafkaVersion *kversion.Versions
	Topics       map[string]TopicOption
	TopicNames   []string
	Timeout      time.Duration
}
