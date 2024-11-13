package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kversion"
	"github.com/twmb/franz-go/pkg/sasl"
	"github.com/twmb/franz-go/pkg/sasl/plain"
	"github.com/twmb/franz-go/pkg/sasl/scram"
)

type (
	TopicConfig struct {
		FromStart bool
	}

	Config struct {
		Sink         []kgo.Opt
		Source       []kgo.Opt
		ClientID     string
		KafkaVersion *kversion.Versions
		Topics       map[string]TopicConfig
	}
)

var (
	sourceBrokers       = flag.String("source-brokers", "", "Comma-separated list of source Kafka brokers")
	sourceTLSEnabled    = flag.Bool("source-tls-enabled", false, "Enable TLS for source Kafka")
	sourceTLSCert       = flag.String("source-tls-cert", "", "TLS certificate for source Kafka")
	sourceTLSInsecure   = flag.Bool("source-tls-insecure", false, "Skip TLS verification for source Kafka")
	sourceSASLEnabled   = flag.Bool("source-sasl-enabled", false, "Enable SASL for source Kafka")
	sourceSASLUsername  = flag.String("source-sasl-username", "", "SASL username for source Kafka")
	sourceSASLPassword  = flag.String("source-sasl-password", "", "SASL password for source Kafka")
	sourceSASLMechanism = flag.String("source-sasl-mechanism", "", "SASL mechanism for source Kafka")
	sourceTimeout       = flag.Duration("source-timeout", 30*time.Second, "Timeout for source Kafka")

	sinkBrokers       = flag.String("sink-brokers", "", "Comma-separated list of sink Kafka brokers")
	sinkTLSEnabled    = flag.Bool("sink-tls-enabled", false, "Enable TLS for sink Kafka")
	sinkTLSCert       = flag.String("sink-tls-cert", "", "TLS certificate for sink Kafka")
	sinkTLSInsecure   = flag.Bool("sink-tls-insecure", false, "Skip TLS verification for sink Kafka")
	sinkSASLEnabled   = flag.Bool("sink-sasl-enabled", false, "Enable SASL for sink Kafka")
	sinkSASLUsername  = flag.String("sink-sasl-username", "", "SASL username for sink Kafka")
	sinkSASLPassword  = flag.String("sink-sasl-password", "", "SASL password for sink Kafka")
	sinkSASLMechanism = flag.String("sink-sasl-mechanism", "", "SASL mechanism for sink Kafka")
	sinkTimeout       = flag.Duration("sink-timeout", 30*time.Second, "Timeout for sink Kafka")

	clientID     = flag.String("client-id", "", "Client ID")
	kafkaVersion = flag.String("kafka-version", "", "Kafka version")
)

func parseFlags() (*Config, error) {
	flag.Parse()

	if kafkaVersion == nil {
		return nil, fmt.Errorf("kafka version is required")
	}

	kVersion := kversion.FromString(*kafkaVersion)
	if kVersion == nil {
		return nil, fmt.Errorf("unknown kafka version %q", *kafkaVersion)
	}

	sourceOpts, err := toFranzOptions(
		sourceBrokers,
		sourceTLSEnabled,
		sourceTLSCert,
		sourceTLSInsecure,
		sourceSASLEnabled,
		sourceSASLUsername,
		sourceSASLPassword,
		sourceSASLMechanism,
		sourceTimeout,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse source options: %w", err)
	}

	sinkOpts, err := toFranzOptions(
		sinkBrokers,
		sinkTLSEnabled,
		sinkTLSCert,
		sinkTLSInsecure,
		sinkSASLEnabled,
		sinkSASLUsername,
		sinkSASLPassword,
		sinkSASLMechanism,
		sinkTimeout,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse sink options: %w", err)
	}

	topics := flag.Args()
	if len(topics) == 0 {
		return nil, fmt.Errorf("topics is empty")
	}

	specifiedTopics := make(map[string]TopicConfig)
	for _, topic := range topics {
		topicOffset := strings.Split(topic, ":")
		if len(topicOffset) != 2 {
			return nil, fmt.Errorf("invalid topic %q", topic)
		}

		var fromStart bool
		switch topicOffset[1] {
		case "from-start", "start", "from-beginning", "beginning":
			fromStart = true
		case "from-end", "end", "from-latest", "latest":
			fromStart = false
		default:
			return nil, fmt.Errorf("invalid offset %q", topicOffset[1])
		}

		specifiedTopics[topicOffset[0]] = TopicConfig{
			FromStart: fromStart,
		}
	}

	clientIDstr := ""
	if clientID != nil {
		clientIDstr = *clientID
	}

	return &Config{
		Sink:         sinkOpts,
		Source:       sourceOpts,
		ClientID:     clientIDstr,
		KafkaVersion: kVersion,
		Topics:       specifiedTopics,
	}, nil
}

func toFranzOptions(
	brokers *string,
	tLSEnabled *bool,
	tLSCert *string,
	tLSInsecure *bool,
	sASLEnabled *bool,
	sASLUsername *string,
	sASLPassword *string,
	sASLMechanism *string,
	timeout *time.Duration,
) ([]kgo.Opt, error) {
	out := make([]kgo.Opt, 0)

	if brokers == nil || *brokers == "" {
		return nil, fmt.Errorf("brokers is empty")
	}
	out = append(out, kgo.SeedBrokers(strings.Split(*brokers, ",")...))

	if tLSEnabled != nil && *tLSEnabled {
		tlsConfig := &tls.Config{
			MinVersion:         tls.VersionTLS12,
			Renegotiation:      tls.RenegotiateFreelyAsClient,
			RootCAs:            x509.NewCertPool(),
			InsecureSkipVerify: tLSInsecure != nil && *tLSInsecure,
		}

		if tLSCert != nil && *tLSCert != "" {
			if ok := tlsConfig.RootCAs.AppendCertsFromPEM([]byte(*tLSCert)); !ok {
				return nil, fmt.Errorf("failed to append cert to root CAs")
			}
		}

		out = append(out, kgo.DialTLSConfig(tlsConfig))
	}

	if sASLEnabled != nil && *sASLEnabled {
		var mechanism sasl.Mechanism
		if sASLMechanism == nil || sASLUsername == nil || sASLPassword == nil {
			return nil, fmt.Errorf("SASL mechanism, username, and password are required")
		}

		switch *sASLMechanism {
		case "plain":
			mechanism = plain.Auth{
				User: *sASLUsername,
				Pass: *sASLPassword,
			}.AsMechanism()
		case "scram-sha-256":
			mechanism = scram.Auth{
				User: *sASLUsername,
				Pass: *sASLPassword,
			}.AsSha256Mechanism()
		case "scram-sha-512":
			mechanism = scram.Auth{
				User: *sASLUsername,
				Pass: *sASLPassword,
			}.AsSha512Mechanism()
		default:
			return nil, fmt.Errorf("unknown SASL mechanism %q", *sASLMechanism)
		}

		out = append(out, kgo.SASL(mechanism))
	}

	if timeout != nil {
		out = append(out, kgo.DialTimeout(*timeout))
	}

	return out, nil
}
